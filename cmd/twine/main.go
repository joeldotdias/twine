package main

import (
	"fmt"
	"os"

	"github.com/joeldotdias/twine/internal/repository"
)

// TODO: doc all the commands
const HELP_TEXT = `usage: gat <command> [<args>...]
`

func main() {
	args := os.Args

	if len(args) < 2 {
		fmt.Fprint(os.Stderr, HELP_TEXT)
	}

	repo, err := repository.Repo(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Malformed repo: %s", err)
	}

	err = repo.Run(args[1:])
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
	}
}
