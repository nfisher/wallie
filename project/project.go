package project

// Backlog is a projects new stories which need sizing or have not been started.
type Backlog struct {
	Project string
	Stories []Story
}

// Sizes returns the size selections.
func (b Backlog) Sizes() []Size {
	return sizes[1:7]
}

// BySize returns a grouping of the backlock by size.
func (b Backlog) BySize() []Group {
	var g []Group

	m := make(map[Size]*Group)
	for i, v := range sizes {
		g = append(g, Group{Name: string(v)})
		m[v] = &g[i]
	}

	for _, v := range b.Stories {
		sg, ok := m[v.Size]
		if !ok {
			sg = m[Unsized]
			sg.Stories = append(sg.Stories, v)
			continue
		}

		sg.Stories = append(sg.Stories, v)
	}

	return g
}

// Group represnts a grouping of stories.
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

const Unsized Size = "To Estimate"

// ExtraSmall is an extra-small story of size 1.
const ExtraSmall Size = "XS"

// Small is a story of size 2.
const Small Size = "S"

// Medium is a story of size 3.
const Medium Size = "M"

// Large is a story of size 5.
const Large Size = "L"

// ExtraLarge is a story of size 8.
const ExtraLarge Size = "XL"

// ExtraExtraLarge is a story of size 13.
const ExtraExtraLarge Size = "XXL"
