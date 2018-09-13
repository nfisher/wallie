package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/nfisher/wallie"
)

// Client is a Jira client.
type CookieClient struct {
	Config  wallie.Config
	Cookies []*http.Cookie
}

// ListStories outputs a list of stories that are not done.
func (c *CookieClient) ListStories(projectID string) (wallie.Stories, error) {
	var stories wallie.Stories

	ss, err := ListIssues(c.Config, projectID, c.Cookies)
	if err != nil {
		return nil, err
	}

	for _, s := range ss {
		story := wallie.Story{
			Description: s.Fields.Description,
			Feature:     s.Fields.Summary,
			Size:        int(s.Fields.StoryPoints),
			ID:          s.Key,
		}
		stories = append(stories, story)
	}

	return stories, nil
}

var client = http.Client{}

func UpdateIssue(config wallie.Config, key, summary, description string, estimate float64, cookies []*http.Cookie) error {
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

func ListIssues(config wallie.Config, projectID string, cookies []*http.Cookie) (Issues, error) {
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
	extraLarge      = 8.0
	extraExtraLarge = 13.0
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
	// rank(10) is to support legacy XL
	return append(i.rank(extraLarge), i.rank(10)...)
}

func (i Issues) ExtraExtraLarge() Issues {
	// rank(20) is to support legacy XXL
	return append(i.rank(extraExtraLarge), i.rank(20)...)
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
