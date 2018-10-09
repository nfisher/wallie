package project_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/nfisher/wallie/project"
)

func Test_render_dialogue(t *testing.T) {
	t.Parallel()

	backlog := project.Backlog{Project: "Wallie"}

	var buf bytes.Buffer
	tpl := project.LoadTemplates()
	err := tpl.ExecuteTemplate(&buf, "story_estimate_dialogue", &backlog)
	if err != nil {
		t.Fatal(err)
	}

	component := buf.String()

	actual := strings.Count(component, `class="column"`)
	if actual != 6 {
		t.Errorf("got count(.column) = %v, want 6", actual)
	}

	actual = strings.Count(component, "<textarea")
	if actual != 1 {
		t.Errorf("got count(textarea) = %v, want 1", actual)
	}

	actual = strings.Count(component, "<input")
	if actual != 2 {
		t.Errorf("got count(input) = %v, want 2", actual)
	}
}

func Test_render_backlog(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	backlog := project.Backlog{Project: "Wallie"}
	tpl := project.LoadTemplates()
	err := tpl.ExecuteTemplate(&buf, "backlog_estimation", &backlog)
	if err != nil {
		t.Fatal(err)
	}

	component := buf.String()

	actual := strings.Count(component, `class="column"`)
	if actual != 7 {
		t.Errorf("got count(.column) = %v, want 7", actual)
	}
}
func Test_render_column(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tpl := project.LoadTemplates()
	group := project.Group{
		Name: "Extra-Extra-Large",
		Stories: []project.Story{
			{
				Author:      "Nathan Fisher",
				Description: "something something something blah blah blah",
				ID:          "ABC-1234",
				Size:        project.ExtraExtraLarge,
				Title:       "Create service skeleton",
			},
		},
	}

	err := tpl.ExecuteTemplate(&buf, "story_group", &group)
	if err != nil {
		t.Fatal(err)
	}

	component := buf.String()

	if !strings.Contains(component, group.Name) {
		t.Errorf("got story group without title, want %v", group.Name)
	}

	if !containsAttr(component, "data-author", "Nathan Fisher") {
		t.Errorf("got story without %s, want %s", "data-author", "Nathan Fisher")
	}
}

func Test_render_story(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tpl := project.LoadTemplates()
	story := project.Story{
		Author:      "Nathan Fisher",
		Description: "something something something blah blah blah",
		ID:          "ABC-1234",
		Size:        project.ExtraExtraLarge,
		Title:       "Create service skeleton",
	}
	err := tpl.ExecuteTemplate(&buf, "story_card", &story)
	if err != nil {
		t.Fatal(err)
	}

	component := buf.String()

	td := []struct {
		name     string
		expected string
	}{
		{"data-author", story.Author},
		{"data-description", story.Description},
		{"data-id", story.ID},
		{"data-size", string(story.Size)},
		{"data-title", story.Title},
	}

	for _, tc := range td {
		t.Run("should include attribute "+tc.name, func(t *testing.T) {
			if !containsAttr(component, tc.name, tc.expected) {
				t.Errorf("got story without %s, want %s", tc.name, tc.expected)
			}
		})
	}

	if strings.Count(component, story.ID) != 2 {
		t.Errorf("got story with %v occurrence of ID, want 2", strings.Count(component, story.ID))
	}

	if strings.Count(component, story.Title) != 2 {
		t.Errorf("got story with %v occurrence of Title, want 2", strings.Count(component, story.Title))
	}
}

func containsAttr(component, key, value string) bool {
	attr := fmt.Sprintf(`%s="%s"`, key, value)
	return strings.Contains(component, attr)
}
