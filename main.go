package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"
)

var (
	Version = ""
	Origin  = ""
)

func main() {
	var configPath string
	var addr string
	var projectID string
	var alwaysReload bool

	log.SetFlags(log.LstdFlags | log.LUTC | log.Lshortfile)
	log.Println("starting wallie")
	log.Println("version:", Version)
	log.Println("source:", Origin)

	port := os.Getenv("PORT")
	if port != "" {
		port = ":" + port
	} else {
		port = ":3000"
	}
	jiraBase := os.Getenv("JIRA_BASE")

	flag.BoolVar(&alwaysReload, "reload", false, "always reload HTML templates")
	flag.StringVar(&configPath, "config", "config.json", "path to the configuration file")
	flag.StringVar(&addr, "listen", port, "listening address")
	flag.StringVar(&projectID, "project", "dmp", "project ID to query on the command-line")
	flag.Parse()

	config, err := readConfig(configPath)
	if err != nil && jiraBase == "" {
		log.Fatal(err)
	}
	if config.SessionName == "" {
		config.SessionName = "JSESSIONID"
	}
	if config.LoginPath == "" {
		config.LoginPath = "/login"
	}
	if jiraBase != "" {
		config.JiraBase = jiraBase
	}

	config.AlwaysReloadHTML = alwaysReload

	mux := http.NewServeMux()

	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, req *http.Request) { io.Copy(w, bytes.NewReader(favIcon)) })

	mux.HandleFunc("/cfd", func(w http.ResponseWriter, req *http.Request) {
		var project = req.URL.Query().Get("project")
		if !validProjectID.MatchString(project) {
			http.Error(w, "unknown project ID", http.StatusNotFound)
			return
		}

		fmt.Fprintln(w, project)
	})

	mux.HandleFunc("/estimation", EstimationHandler(config))

	mux.HandleFunc(config.LoginPath, Login(config))

	log.Printf("binding to %s", addr)
	log.Fatal(http.ListenAndServe(addr, LogRequests(RequireLogin(mux, config))))
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func RequireLogin(h http.Handler, config Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if config.LoginPath == req.URL.EscapedPath() {
			h.ServeHTTP(w, req)
			return
		}

		_, err := req.Cookie(config.SessionName)
		if err != nil {
			// override method so that it forces form rendering
			req.Method = http.MethodGet
			Login(config)(w, req)
			return
		}

		h.ServeHTTP(w, req)
	})
}

func Login(config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost && req.URL.EscapedPath() == config.LoginPath {
			err := req.ParseForm()
			if err != nil {
				http.Error(w, "unable to parse form values", http.StatusBadRequest)
				return
			}

			username := req.FormValue("email")
			password := req.FormValue("password")

			loginRequest := LoginRequest{
				Username: username,
				Password: password,
			}

			b, err := json.Marshal(&loginRequest)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			authReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/rest/auth/1/session", config.JiraBase), bytes.NewBuffer(b))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			authReq.Header.Add("Content-Type", "application/json")

			resp, err := client.Do(authReq)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer resp.Body.Close()

			io.Copy(os.Stdout, resp.Body)

			for _, c := range resp.Cookies() {
				c.MaxAge = 60 * 60 // 1 hour
				http.SetCookie(w, c)
			}

			redirectCookie, err := req.Cookie("wallieRedirect")
			if err != nil {
				http.Error(w, err.Error(), http.StatusNoContent)
			}

			http.SetCookie(w, &http.Cookie{Name: "wallieRedirect", MaxAge: -1})
			tpl.ExecuteTemplate(w, "login_redirect", redirectCookie.Value)
			return
		}

		_, cookieErr := req.Cookie("wallieRedirect")
		if cookieErr == http.ErrNoCookie {
			redirectCookie := &http.Cookie{
				Name:     "wallieRedirect",
				Value:    fmt.Sprintf("%s?%s", req.URL.EscapedPath(), req.URL.Query().Encode()),
				HttpOnly: true,
			}
			http.SetCookie(w, redirectCookie)
		}

		err := tpl.ExecuteTemplate(w, "login", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func LogRequests(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		wr := &ResponseWriter{ResponseWriter: w}
		start := time.Now()
		h.ServeHTTP(wr, req)
		log.Printf(`%v %s %s - %v - %v - %vB`, wr.Status(), req.Method, req.URL.Path, req.RemoteAddr, time.Now().Sub(start), wr.Bytes())
	})
}

type ResponseWriter struct {
	http.ResponseWriter
	bytes       int
	status      int
	wroteHeader bool
}

func (w *ResponseWriter) Status() int {
	return w.status
}

func (w *ResponseWriter) Bytes() int {
	return w.bytes
}

func (w *ResponseWriter) Write(p []byte) (n int, err error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	w.bytes += len(p)

	return w.ResponseWriter.Write(p)
}

func (w *ResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
	// Check after in case there's error handling in the wrapped ResponseWriter.
	if w.wroteHeader {
		return
	}
	w.status = code
	w.wroteHeader = true
}

type Config struct {
	JiraBase         string
	LoginPath        string
	SessionName      string
	AlwaysReloadHTML bool `json:"-"`
}

func readConfig(path string) (Config, error) {
	var config Config

	r, err := os.Open(path)
	if err != nil {
		return config, err
	}

	err = json.NewDecoder(r).Decode(&config)
	if err != nil {
		return config, err
	}

	return config, nil
}

var validProjectID = regexp.MustCompile(`^\w+$`)

func EstimationHandler(config Config) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		var tpl = tpl

		projectID := req.URL.Query().Get("project")

		if !validProjectID.MatchString(projectID) {
			http.Error(w, "Invalid project ID", http.StatusNotFound)
			return
		}

		if config.AlwaysReloadHTML {
			tpl = template.Must(template.ParseGlob("tpl/*.html"))
		}

		if req.Method == http.MethodPost {
			err := req.ParseForm()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			key := req.FormValue("key")
			if !validKey.MatchString(key) {
				http.Error(w, "invalid key!", http.StatusBadRequest)
				return
			}
			summary := req.FormValue("summary")
			description := req.FormValue("description")
			estimate := tee2estimate(req.FormValue("size"))

			err = UpdateIssue(config, key, summary, description, estimate, req.Cookies())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		issues, err := ListIssues(config, projectID, req.Cookies())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = tpl.ExecuteTemplate(w, "estimation_board", EstimationPage{JiraBase: config.JiraBase, Issues: issues})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

type EstimationPage struct {
	JiraBase string
	Issues   Issues
}

var tpl = template.Must(template.ParseGlob("tpl/*.html"))

var client = http.Client{}

func UpdateIssue(config Config, key, summary, description string, estimate float64, cookies []*http.Cookie) error {
	updateRequest := UpdateIssueRequest{
		Fields: IssueFields{
			Summary:     summary,
			Description: description,
			StoryPoints: estimate,
		},
	}

	b, err := json.Marshal(updateRequest)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/rest/api/2/issue/%s", config.JiraBase, key), bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		b, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("%d => %s\n", resp.StatusCode, b)
	}

	return nil
}

type UpdateIssueRequest struct {
	Fields IssueFields `json:"fields"`
}

func ListIssues(config Config, projectID string, cookies []*http.Cookie) (Issues, error) {
	client := http.Client{}

	searchRequest := SearchRequest{
		JQL:        fmt.Sprintf(`type = Story AND project = "%s" AND status not in (Done, Closed) ORDER BY rank`, projectID),
		MaxResults: 100,
		Fields: []string{
			"summary",
			"customfield_10006",
			"description",
			"reporter",
		},
	}

	b, err := json.Marshal(searchRequest)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/rest/api/2/search", config.JiraBase), bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	for _, c := range cookies {
		req.AddCookie(c)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %v", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var queryResp QueryResp
	err = json.Unmarshal(body, &queryResp)
	if err != nil {
		log.Printf("%s\n", body)
		return nil, err
	}

	return queryResp.Issues, nil
}

type QueryResp struct {
	MaxResults int    `json:"maxResults"`
	Total      int    `json:"total"`
	StartAt    int    `json:"startAt"`
	Issues     Issues `json:"issues"`
}

var validKey = regexp.MustCompile(`^[A-Z]+-[0-9]+$`)

func tee2estimate(size string) float64 {
	switch size {
	case "XS":
		return extraSmall
	case "S":
		return small
	case "M":
		return medium
	case "L":
		return large
	case "XL":
		return extraLarge
	case "XXL":
		return extraExtraLarge
	case "U":
		return unknown
	}

	return undefined
}

type Issues []Issue

func (i Issues) rank(r float64) Issues {
	var issues Issues
	for _, v := range i {
		if r == 0.0 && v.Fields.StoryPoints < 1.0 {
			issues = append(issues, v)
		} else if v.Fields.StoryPoints == r {
			issues = append(issues, v)
		}
	}
	return issues
}

// XS, S, M, L, XL, XXL
// 1,  2, 3, 5, 10, 20

const (
	unknown         = -1.0
	undefined       = 0.0
	extraSmall      = 1.0
	small           = 2.0
	medium          = 3.0
	large           = 5.0
	extraLarge      = 10.0
	extraExtraLarge = 20.0
)

func (i Issues) ExtraSmall() Issues {
	return i.rank(extraSmall)
}

func (i Issues) Small() Issues {
	return i.rank(small)
}

func (i Issues) Medium() Issues {
	return i.rank(medium)
}

func (i Issues) Large() Issues {
	return i.rank(large)
}

func (i Issues) ExtraLarge() Issues {
	return i.rank(extraLarge)
}

func (i Issues) ExtraExtraLarge() Issues {
	return i.rank(extraExtraLarge)
}

func (i Issues) Unknown() Issues {
	return i.rank(-1.0)
}

func (i Issues) Other() Issues {
	return i.rank(0.0)
}

type Issue struct {
	Key    string      `json:"key"`
	Self   string      `json:"self"`
	Fields IssueFields `json:"fields"`
}

type IssueFields struct {
	Summary     string    `json:"summary"`
	Description string    `json:"description"`
	StoryPoints float64   `json:"customfield_10006,omitempty"`
	Reporter    *Reporter `json:"reporter,omitempty"`
}

type Reporter struct {
	DisplayName string `json:"displayName"`
}

type SearchRequest struct {
	JQL        string   `json:"jql"`
	StartAt    int      `json:"startAt"`
	MaxResults int      `json:"maxResults"`
	Fields     []string `json:"fields"`
}

var favIcon = []byte{
	0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x10, 0x10, 0x02, 0x00, 0x01, 0x00, 0x01, 0x00, 0xb0, 0x00,
	0x00, 0x00, 0x16, 0x00, 0x00, 0x00, 0x28, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x20, 0x00,
	0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0x00, 0x00, 0xff, 0xff, 0x00, 0x00, 0xff, 0xff,
	0x00, 0x00, 0xff, 0xff, 0x00, 0x00, 0xff, 0xff, 0x00, 0x00, 0xff, 0xff, 0x00, 0x00, 0xff, 0xff,
	0x00, 0x00, 0xff, 0xff, 0x00, 0x00,
}
