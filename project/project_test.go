package project_test

import (
	"testing"

	"github.com/nfisher/wallie/project"
)

func Test_BySize(t *testing.T) {
	t.Parallel()
	// reverse order backlog to test that BySize reorders groups.
	backlog := project.Backlog{
		Stories: []project.Story{
			{Title: "XXL Stories", Size: project.ExtraExtraLarge},
			{Title: "XL Stories", Size: project.ExtraLarge},
			{Title: "L Stories", Size: project.Large},
			{Title: "M Stories", Size: project.Medium},
			{Title: "S Stories", Size: project.Small},
			{Title: "XS Stories", Size: project.ExtraSmall},
			{Title: "Unestimated Stories", Size: project.Unsized},
		},
	}
	g := backlog.BySize()

	if len(g) != 7 {
		t.Errorf("got len = %v, want 7", len(g))
	}

	td := []struct {
		name string
		size project.Size
	}{
		{"Unestimated Stories", project.Unsized},
		{"XS Stories", project.ExtraSmall},
		{backlog.Stories[4].Title, project.Small},
		{backlog.Stories[3].Title, project.Medium},
		{backlog.Stories[2].Title, project.Large},
		{backlog.Stories[1].Title, project.ExtraLarge},
		{backlog.Stories[0].Title, project.ExtraExtraLarge},
	}

	for i, tc := range td {
		t.Run(tc.name, func(t *testing.T) {
			actual := g[i]
			expected := string(tc.size)
			if expected != actual.Name {
				t.Errorf("want group name %v = %v, got %v", i, expected, actual.Name)
			}
		})
	}
}
