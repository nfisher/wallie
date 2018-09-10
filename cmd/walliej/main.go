package main

import (
	"log"
	"os"

	"github.com/nfisher/wallie/jira"
)

var (
	// Version is the git SHA injected at the time of compilation.
	Version = ""

	// Origin is the git origin injected at the time of compilation.
	Origin = ""
)

func main() {
	err := jira.Execute(Version, Origin)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
