package project

type Client interface {
	ListStories(projectID string) (Backlog, error)
	UpdateStory(projectID, id, title, description, size string) error
}

// Backlog is a projects new stories which need sizing or are not done.
type Backlog struct {
	Project string
	Stories []Story
	BaseURL string
}

// Sizes returns the available tee-shirt sizes.
func (b Backlog) Sizes() []Size {
	return sizes[1:]
}

// Count returns the count of stories not done.
func (b Backlog) Count() int {
	return len(b.Stories)
}

// BySize returns a grouping of the backlock by size.
func (b Backlog) BySize() []*Group {
	var gg []*Group

	m := make(map[Size]*Group)
	for _, v := range sizes {
		g := &Group{
			Name: string(v),
		}
		gg = append(gg, g)
		m[v] = g
	}

	for _, v := range b.Stories {
		sg, ok := m[v.Size]
		if !ok {
			sg = m[Unsized]
		}
		sg.Stories = append(sg.Stories, v)
	}

	return gg
}

// Group represents a grouping of stories.
type Group struct {
	Name    string
	Stories []Story
}

// Story encapsulates all of the core data related to a story.
type Story struct {
	Author      string
	Description string
	ID          string
	Size        Size
	Title       string
}

// Size is a story size type.
type Size string

var sizes = []Size{Unsized, ExtraSmall, Small, Medium, Large, ExtraLarge, ExtraExtraLarge}

const (
	// Unsized is a story of an unknown size.
	Unsized Size = "To Estimate"

	// ExtraSmall is an extra-small story of size 1.
	ExtraSmall Size = "XS"

	// Small is a story of size 2.
	Small Size = "S"

	// Medium is a story of size 3.
	Medium Size = "M"

	// Large is a story of size 5.
	Large Size = "L"

	// ExtraLarge is a story of size 8.
	ExtraLarge Size = "XL"

	// ExtraExtraLarge is a story of size 13.
	ExtraExtraLarge Size = "XXL"
)
