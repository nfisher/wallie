package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/nfisher/wallie"
	"github.com/nfisher/wallie/jira"
	"github.com/nfisher/wallie/reqlog"
)

var (
	// Version is the git SHA injected at the time of compilation.
	Version = ""

	// Origin is the git origin injected at the time of compilation.
	Origin = ""
)

func main() {
	err := Execute()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func Execute() error {
	var configPath string
	var addr string
	var projectID string
	var alwaysReload bool
	var port = DefaultAddress()
	var jiraBase = os.Getenv("JIRA_BASE")

	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.LUTC | log.Lshortfile)
	log.Println("starting wallie")
	log.Println("version:", Version)
	log.Println("source:", Origin)

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

	config.AlwaysReloadHTML = alwaysReload

	mux := http.NewServeMux()

	mux.HandleFunc("/favicon.ico", jira.Favicon)
	mux.HandleFunc("/cfd", jira.CumulativeFlow)
	mux.HandleFunc("/estimation", jira.EstimationHandler(config))
	mux.HandleFunc("/sizing", jira.SizingHandler(config))
	mux.HandleFunc(config.LoginPath, jira.Login(config))

	log.Printf("binding to %s", addr)
	return http.ListenAndServe(addr, reqlog.LogRequests(jira.RequireLogin(mux, config)))
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
