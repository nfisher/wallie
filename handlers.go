package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
)

func CumulativeFlow(w http.ResponseWriter, req *http.Request) {
	var project = req.URL.Query().Get("project")
	if !validProjectID.MatchString(project) {
		http.Error(w, "unknown project ID", http.StatusNotFound)
		return
	}

	fmt.Fprintln(w, project)
}

func Favicon(w http.ResponseWriter, req *http.Request) { io.Copy(w, bytes.NewReader(favIcon)) }

func RequireLogin(h http.Handler, config Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		p := req.URL.EscapedPath()
		if config.LoginPath == p || "/favicon.ico" == p {
			h.ServeHTTP(w, req)
			return
		}

		_, err := req.Cookie(config.SessionName)
		if err != nil {
			// override method so that it forces form rendering
			req.Method = http.MethodGet
			Login(config)(w, req)
			return
		}

		h.ServeHTTP(w, req)
	})
}

func Login(config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost && req.URL.EscapedPath() == config.LoginPath {
			err := req.ParseForm()
			if err != nil {
				http.Error(w, "unable to parse form values", http.StatusBadRequest)
				return
			}

			username := req.FormValue("email")
			password := req.FormValue("password")

			loginRequest := LoginRequest{
				Username: username,
				Password: password,
			}

			b, err := json.Marshal(&loginRequest)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			authReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/rest/auth/1/session", config.JiraBase), bytes.NewBuffer(b))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			authReq.Header.Add("Content-Type", "application/json")

			resp, err := client.Do(authReq)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer resp.Body.Close()

			for _, c := range resp.Cookies() {
				c.MaxAge = 60 * 60 // 1 hour
				http.SetCookie(w, c)
			}

			redirectCookie, err := req.Cookie("wallieRedirect")
			if err != nil {
				http.Error(w, err.Error(), http.StatusNoContent)
			}

			http.SetCookie(w, &http.Cookie{Name: "wallieRedirect", MaxAge: -1})
			tpl.ExecuteTemplate(w, "login_redirect", redirectCookie.Value)
			return
		}

		_, cookieErr := req.Cookie("wallieRedirect")
		if cookieErr == http.ErrNoCookie {
			redirectCookie := &http.Cookie{
				Name:     "wallieRedirect",
				Value:    fmt.Sprintf("%s?%s", req.URL.EscapedPath(), req.URL.Query().Encode()),
				HttpOnly: true,
			}
			http.SetCookie(w, redirectCookie)
		}

		err := tpl.ExecuteTemplate(w, "login", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// LoginRequest encapsulates a user login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func SizingHandler(config Config) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		var tpl = tpl

		projectID := req.URL.Query().Get("project")

		if !validProjectID.MatchString(projectID) {
			http.Error(w, "Invalid project ID", http.StatusNotFound)
			return
		}

		if config.AlwaysReloadHTML {
			tpl = template.Must(template.ParseGlob("tpl/*.html"))
		}

		issues, err := ListIssues(config, projectID, req.Cookies())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = tpl.ExecuteTemplate(w, "sizing_board", EstimationPage{JiraBase: config.JiraBase, Issues: issues})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func EstimationHandler(config Config) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		var tpl = tpl

		projectID := req.URL.Query().Get("project")

		if !validProjectID.MatchString(projectID) {
			http.Error(w, "Invalid project ID", http.StatusNotFound)
			return
		}

		if config.AlwaysReloadHTML {
			tpl = template.Must(template.ParseGlob("tpl/*.html"))
		}

		if req.Method == http.MethodPost {
			err := req.ParseForm()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			key := req.FormValue("key")
			if !validKey.MatchString(key) {
				http.Error(w, "invalid key!", http.StatusBadRequest)
				return
			}
			summary := req.FormValue("summary")
			description := req.FormValue("description")
			estimate := tee2estimate(req.FormValue("size"))

			err = UpdateIssue(config, key, summary, description, estimate, req.Cookies())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		issues, err := ListIssues(config, projectID, req.Cookies())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = tpl.ExecuteTemplate(w, "estimation_board", EstimationPage{JiraBase: config.JiraBase, Issues: issues})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

var favIcon = []byte{
	0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x10, 0x10, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0xb0, 0x00,
	0x00, 0x00, 0x16, 0x00, 0x00, 0x00, 0x28, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x20, 0x00,
	0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x40, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0xfb, 0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0xc0, 0x00, 0x00, 0x0f, 0xf0,
	0x00, 0x00, 0x1c, 0x38, 0x00, 0x00, 0x3b, 0xdc, 0x00, 0x00, 0x37, 0xec, 0x00, 0x00, 0x77, 0xee,
	0x00, 0x00, 0x7f, 0xfe, 0x00, 0x00, 0x79, 0x9e, 0x00, 0x00, 0x79, 0x9e, 0x00, 0x00, 0x3f, 0xfc,
	0x00, 0x00, 0x3f, 0xfc, 0x00, 0x00, 0x1f, 0xf8, 0x00, 0x00, 0x0f, 0xf0, 0x00, 0x00, 0x03, 0xc0,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xfc, 0x3f, 0x00, 0x00, 0xf0, 0x0f, 0x00, 0x00, 0xe0, 0x07,
	0x00, 0x00, 0xc0, 0x03, 0x00, 0x00, 0x80, 0x01, 0x00, 0x00, 0x80, 0x01, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80, 0x01,
	0x00, 0x00, 0x80, 0x01, 0x00, 0x00, 0xc0, 0x03, 0x00, 0x00, 0xe0, 0x07, 0x00, 0x00, 0xf0, 0x0f,
	0x00, 0x00, 0xfc, 0x3f, 0x00, 0x00,
}
