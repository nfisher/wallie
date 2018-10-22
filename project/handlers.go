package project

import (
	"net/http"

	"github.com/nfisher/wallie"
)

func FlowHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		tmpl := LoadTemplates(true)
		projectID := req.URL.Query().Get("project")

		err := tmpl.ExecuteTemplate(w, "story_flow_head", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		flusher, ok := w.(http.Flusher)
		if ok {
			flusher.Flush()
		}

		var contents = struct {
			Project string
			Count   int
		}{
			Project: projectID,
		}

		err = tmpl.ExecuteTemplate(w, "story_flow_content", &contents)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// TshirtHandler handles estimation for individual stories with examples for each tee-shirt size where available.
func TshirtHandler(fn func(wallie.Config, []*http.Cookie) Client, config wallie.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		tmpl := LoadTemplates(config.AlwaysReloadHTML)
		client := fn(config, req.Cookies())
		projectID := req.URL.Query().Get("project")

		err := tmpl.ExecuteTemplate(w, "story_estimation_head", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		flusher, ok := w.(http.Flusher)
		if ok {
			flusher.Flush()
		}

		if req.Method == http.MethodPost {
			err := req.ParseForm()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			id := req.FormValue("id")
			title := req.FormValue("title")
			description := req.FormValue("description")
			size := req.FormValue("size")

			err = client.UpdateStory(projectID, id, title, description, size)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		backlog, err := client.ListStories(projectID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = tmpl.ExecuteTemplate(w, "story_estimation_content", &backlog)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
