package main

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC | log.Lshortfile)
	var printBacklog bool
	var configPath string
	var addr string
	var projectID string
	var alwaysReload bool


	flag.BoolVar(&printBacklog, "backlog", false, "print backlog and exit")
	flag.BoolVar(&alwaysReload, "reload", false, "always reload HTML templates")
	flag.StringVar(&configPath, "config", "config.json", "path to the configuration file")
	flag.StringVar(&addr, "listen", ":3000", "listening address")
	flag.StringVar(&projectID, "project", "dmp", "project ID to query on the command-line")
	flag.Parse()

	config, err := readConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}
	config.AlwaysReloadHTML = alwaysReload

	if printBacklog {
		PrintBacklog(config, projectID)
		return
	}

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(w, "projects")
	})

	http.HandleFunc("/cfd", func(w http.ResponseWriter, req *http.Request) {
		var project = req.URL.Query().Get("project")
		if !validProjectID.MatchString(project) {
			http.Error(w, "unknown project ID", http.StatusNotFound)
			return
		}

		fmt.Fprintln(w, project)
	})

	http.HandleFunc("/estimation", BasicAuth(ProjectHandler(config), config.BasicUsername, config.BasicPassword, "Authentication Required"))

	log.Printf("binding to %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

type Config struct {
	JiraBase         string
	Username         string `json:"username"`
	Password         string `json:"password"`
	BasicUsername    string `json:"basicUsername"`
	BasicPassword    string `json:"basicPassword"`
	AlwaysReloadHTML bool	`json:"-"`
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

func ProjectHandler(config Config) func(w http.ResponseWriter, req *http.Request) {
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

			err = UpdateIssue(config, key, summary, description, estimate)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		issues, err := ListIssues(config, projectID)
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

func BasicAuth(handler http.HandlerFunc, username, password, realm string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()

		if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
			w.WriteHeader(401)
			w.Write([]byte("Unauthorised.\n"))
			return
		}

		handler(w, r)
	}
}

var tpl = template.Must(template.ParseGlob("tpl/*.html"))

var client = http.Client{}

func UpdateIssue(config Config, key, summary, description string, estimate float64) error {
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
	req.SetBasicAuth(config.Username, config.Password)

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

func ListIssues(config Config, projectID string) (Issues, error) {
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
	req.SetBasicAuth(config.Username, config.Password)

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
	case "S":
		return 1.0
	case "M":
		return 2.0
	case "L":
		return 3.0
	case "U":
		return -1.0
	}

	return 0.0
}

func PrintBacklog(config Config, projectID string) {
	issues, err := ListIssues(config, projectID)
	if err != nil {
		log.Fatal(err)
	}

	for _, issue := range issues {
		fmt.Printf("[%-12s] %s (%v)\n", issue.Key, issue.Fields.Summary, issue.Fields.StoryPoints)
	}
}

type Issues []Issue

func (i Issues) rank(r float64) Issues {
	var issues Issues
	for _, v := range i {
		if v.Fields.StoryPoints == r {
			issues = append(issues, v)
		}
	}
	return issues
}

func (i Issues) Small() Issues {
	return i.rank(1.0)
}

func (i Issues) Medium() Issues {
	return i.rank(2.0)
}

func (i Issues) Large() Issues {
	return i.rank(3.0)
}

func (i Issues) ExtraLarge() Issues {
	return i.rank(5.0)
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
