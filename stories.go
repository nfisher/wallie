package wallie

type Stories []Story

// Story contains
type Story struct {
	Feature     string
	Description string
	Author      string
	ID          string
	Self        string
	Size        int
}
