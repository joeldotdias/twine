package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joeldotdias/twine/internal/helpers"
)

type Repository struct {
	worktree string
	gitDir   string
	conf     Config
	refs     map[string]string
}

func Repo(cmd string) (*Repository, error) {
	var worktree string
	var err error

	if cmd == "init" {
		worktree, err = os.Getwd()
	} else {
		worktree, err = findRepoRoot(".")
	}
	if err != nil {
		return nil, err
	}

	gitDir := filepath.Join(worktree, ".git")
	conf := makeCfg()
	refs := make(map[string]string)

	return &Repository{
		worktree,
		gitDir,
		conf,
		refs,
	}, nil
}

func (repo *Repository) makePath(paths ...string) string {
	parts := append([]string{repo.gitDir}, paths...)
	return filepath.Join(parts...)
}

func (repo *Repository) Run(args []string) error {
	cmd := args[0]

	switch cmd {
	case "init":
		return repo.Init()
	case "cat-file":
		return repo.CatFile(args[1], args[2])
	case "hash-object":
		objKind := "blob"
		write := false
		return repo.HashObject(write, objKind, args[1])
	}

	return fmt.Errorf("%s command wasn't found", cmd)
}

func findRepoRoot(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	if helpers.IsDir(filepath.Join(absPath, ".git")) {
		return absPath, nil
	}

	parent := filepath.Dir(absPath)
	if parent == absPath {
		return "", errors.New("Not in a repository. Run gat init to make one.")
	}

	return findRepoRoot(parent)
}
