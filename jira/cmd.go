package jira

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/nfisher/wallie"
	"github.com/nfisher/wallie/project"
	"github.com/nfisher/wallie/reqlog"
)

func Execute(version, origin string) error {
	var configPath string
	var addr string
	var projectID string
	var alwaysReload bool
	var isInsecure bool
	var port = DefaultAddress()
	var jiraBase = os.Getenv("JIRA_BASE")

	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.LUTC | log.Lshortfile)
	log.Println("starting wallie")
	log.Println("version:", version)
	log.Println("source:", origin)

	flag.BoolVar(&isInsecure, "insecure", false, "local development insecure cookies")
	flag.BoolVar(&alwaysReload, "reload", false, "always reload HTML templates")
	flag.StringVar(&configPath, "config", "config.json", "path to the configuration file")
	flag.StringVar(&addr, "listen", port, "listening address")
	flag.StringVar(&projectID, "project", "dmp", "project ID to query on the command-line")
	flag.Parse()

	config, err := readConfig(configPath)
	if err != nil && jiraBase == "" {
		return err
	}
	if config.SessionName == "" {
		config.SessionName = "JSESSIONID"
	}
	if config.LoginPath == "" {
		config.LoginPath = "/login"
	}
	if jiraBase != "" {
		config.JiraBase = jiraBase
	}
	if isInsecure {
		config.IsInsecure = true
		log.Println("overriding secure cookies")
	}

	config.AlwaysReloadHTML = alwaysReload

	mux := http.NewServeMux()

	mux.HandleFunc("/favicon.ico", Favicon)

	mux.HandleFunc("/tshirt", project.TshirtHandler(New, config))
	mux.HandleFunc("/flow", project.FlowHandler())

	mux.HandleFunc("/estimation", project.TshirtHandler(New, config))
	mux.HandleFunc("/sizing", SizingHandler(config))
	mux.HandleFunc(config.LoginPath, Login(config))

	log.Printf("binding to %s", addr)
	return http.ListenAndServe(addr, reqlog.LogRequests(RequireLogin(mux, config)))
}

func readConfig(path string) (wallie.Config, error) {
	var config wallie.Config

	r, err := os.Open(path)
	if err != nil {
		return config, err
	}

	err = json.NewDecoder(r).Decode(&config)
	if err != nil {
		return config, err
	}

	return config, nil
}

// DefaultAddress returns `:$PORT` if defined else `:3000`.
func DefaultAddress() string {
	port := os.Getenv("PORT")

	var listeningAddress = ":3000"

	if port != "" {
		listeningAddress = ":" + port
	}

	return listeningAddress
}
