package project

import (
	"html/template"
	"sync"
)

type templates struct {
	templates *template.Template
	sync.RWMutex
	isLoaded bool
}

var tpl templates

// LoadTemplates loads the html templates associated with this package.
func LoadTemplates(alwaysReload bool) *template.Template {
	tpl.RLock()
	isLoaded := tpl.isLoaded
	t := tpl.templates
	tpl.RUnlock()

	if !alwaysReload && isLoaded {
		return t
	}

	tpl.Lock()
	defer tpl.Unlock()
	if !alwaysReload && isLoaded {
		t := tpl.templates
		return t
	}

	tpl.templates = template.Must(template.ParseFiles("tpl/project.html"))
	tpl.isLoaded = true
	t = tpl.templates

	return t
}
