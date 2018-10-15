package project_test

import (
	"reflect"
	"testing"

	"github.com/nfisher/wallie/project"
)

func Test_Sizes(t *testing.T) {
	sizes := project.Backlog{}.Sizes()

	expected := []project.Size{
		project.ExtraSmall,
		project.Small,
		project.Medium,
		project.Large,
		project.ExtraLarge,
		project.ExtraExtraLarge,
	}

	if !reflect.DeepEqual(sizes, expected) {
		t.Errorf("got Sizes() = %v, want %v", sizes, expected)
	}
}

func Test_BySize_XS(t *testing.T) {
	t.Parallel()

	gg := project.Backlog{
		Stories: []project.Story{
			{Title: "XS Stories", Size: project.ExtraSmall},
		},
	}.BySize()

	if len(gg) != 7 {
		t.Fatalf("got len = %v, want 7", len(gg))
	}

	if len(gg[1].Stories) != 1 {
		t.Errorf("got %#v, want len = 1", gg[1])
	}
}

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
			{Title: "Unestimated Stories"},
		},
	}
	g := backlog.BySize()

	if len(g) != 7 {
		t.Fatalf("got len = %v, want 7", len(g))
	}

	td := []struct {
		name string
		size project.Size
	}{
		{"Unestimated Stories", project.Unsized},
		{"XS Stories", project.ExtraSmall},
		{"S Stories", project.Small},
		{"M Stories", project.Medium},
		{"L Stories", project.Large},
		{"XL Stories", project.ExtraLarge},
		{"XXL Stories", project.ExtraExtraLarge},
	}

	for i, tc := range td {
		t.Run(tc.name, func(t *testing.T) {
			actual := g[i]
			expected := string(tc.size)
			if expected != actual.Name {
				t.Errorf("got group name [%v] = %v, want %v", i, actual.Name, expected)
			}

			if len(actual.Stories) != 1 {
				t.Errorf("got group len(%v) = %v, want 1", i, len(actual.Stories))
			}
		})
	}
}
