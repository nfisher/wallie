package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/nfisher/wallie/reqlog"
)

var (
	// Version is the git SHA injected at the time of compilation.
	Version = ""

	// Origin is the git origin injected at the time of compilation.
	Origin = ""
)

func main() {
	err := Execute()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func Execute() error {
	var configPath string
	var addr string
	var projectID string
	var alwaysReload bool

	log.SetFlags(log.LstdFlags | log.LUTC | log.Lshortfile)
	log.Println("starting wallie")
	log.Println("version:", Version)
	log.Println("source:", Origin)

	port := DefaultAddress()
	jiraBase := os.Getenv("JIRA_BASE")

	flag.BoolVar(&alwaysReload, "reload", false, "always reload HTML templates")
	flag.StringVar(&configPath, "config", "config.json", "path to the configuration file")
	flag.StringVar(&addr, "listen", port, "listening address")
	flag.StringVar(&projectID, "project", "dmp", "project ID to query on the command-line")
	flag.Parse()

	config, err := readConfig(configPath)
	if err != nil && jiraBase == "" {
		return err
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

	mux.HandleFunc("/favicon.ico", Favicon)
	mux.HandleFunc("/cfd", CumulativeFlow)
	mux.HandleFunc("/estimation", EstimationHandler(config))
	mux.HandleFunc("/sizing", SizingHandler(config))
	mux.HandleFunc(config.LoginPath, Login(config))

	log.Printf("binding to %s", addr)
	return http.ListenAndServe(addr, reqlog.LogRequests(RequireLogin(mux, config)))
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

type Config struct {
	JiraBase         string
	LoginPath        string
	SessionName      string
	AlwaysReloadHTML bool `json:"-"`
}

// DefaultAddress returns `:$PORT` if defined else `:3000`.
func DefaultAddress() string {
	port := os.Getenv("PORT")

	var listeningAddress = ":3000"

	if port != "" {
		listeningAddress = ":" + port
	}

	return listeningAddress
}

var validProjectID = regexp.MustCompile(`^\w+$`)

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

		return fmt.Errorf("%d => %s", resp.StatusCode, b)
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

	log.Printf("read %v issues, max %v", queryResp.Total, queryResp.MaxResults)

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
