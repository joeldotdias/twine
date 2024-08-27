package main

import (
	"fmt"
	"os"

	"github.com/joeldotdias/twine/internal/repository"
)

func main() {
	args := os.Args

	if len(args) < 2 {
		fmt.Fprint(os.Stderr, HELP_TEXT)
		os.Exit(1)
	}

	repo, err := repository.Repo(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Malformed repo: %s", err)
	}

	err = repo.Run(args[1:])
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
}

// TODO: Make this text better
const HELP_TEXT = `usage: gat <command> [<args>...]

	Commands:
	init         Initialize a new, empty repository

	cat-file     Provide content or type and size information for repository objects
	cat-file (-s | -t | -p) <object> | cat-file <type> <object>
	-s		size of the <object>
	-t 		type of the <object>
	-p 		pretty print the serialized <object>

	hash-object  Compute object ID and optionally create a blob from a file
	hash-object [-w] <files>...
	-w 		write object into database

	ls-tree      List the contents of a tree object
	ls-tree [-r] <tree-ish>
	-r 		recurse into sub-trees

	log          Show commit logs
`
