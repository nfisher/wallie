package project

import (
	"net/http"

	"github.com/nfisher/wallie"
)

func BacklogEstimation(fn func(wallie.Config, []*http.Cookie) Client, config wallie.Config) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		tpl := LoadTemplates()
		cookies := req.Cookies()
		client := fn(config, cookies)
		projectID := req.URL.Query().Get("project")

		err := tpl.ExecuteTemplate(w, "story_estimation_head", nil)
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

		err = tpl.ExecuteTemplate(w, "story_estimation_content", &backlog)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
