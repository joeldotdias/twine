package repository

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/joeldotdias/twine/internal/helpers"
)

type Repository struct {
	worktree string
	gitDir   string
	conf     Config
	refStore *RefStore
}

type RefStore struct {
	heads map[string]string
	tags  map[string]string
}

func Repo(cmd string) (*Repository, error) {
	var worktree string
	var err error
	// bad bad code
	isInit := cmd == "init"

	if isInit {
		worktree, err = os.Getwd()
	} else {
		worktree, err = helpers.SearchRoot(".")
	}
	if err != nil {
		return nil, err
	}

	gitDir := filepath.Join(worktree, ".git")
	conf := makeCfg()
	reef := &RefStore{}

	repo := &Repository{
		worktree,
		gitDir,
		conf,
		reef,
	}

	if !isInit {
		err = repo.findRefs()
		if err != nil {
			return nil, err
		}
	}

	return repo, nil
}

func (repo *Repository) makePath(paths ...string) string {
	parts := append([]string{repo.gitDir}, paths...)
	return filepath.Join(parts...)
}

// loads the refs and current head
// into repo struct
func (repo *Repository) findRefs() error {
	repo.refStore.heads = make(map[string]string)
	repo.refStore.tags = make(map[string]string)

	paths := []struct {
		path    string
		refType string
	}{
		{repo.makePath("refs", "heads"), "heads"},
		{repo.makePath("refs", "tags"), "tags"},
	}

	for _, p := range paths {
		err := filepath.WalkDir(p.path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				contents, err := os.ReadFile(path)
				if err != nil {
					return err
				}

				ref := string(contents[:len(contents)-1])
				name := filepath.Base(path)
				switch p.refType {
				case "heads":
					repo.refStore.heads[ref] = name
				case "tags":
					repo.refStore.tags[ref] = name
				}
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error walking %s: %w", p.path, err)
		}
	}

	return nil
}

func (repo *Repository) Run(args []string) error {
	cmd := args[0]

	switch cmd {
	case "init":
		return repo.Init()

	case "cat-file":
		return repo.CatFile(args[1:])

	case "hash-object":
		hashObjectCmd := flag.NewFlagSet("hash-object", flag.ExitOnError)
		objKind := hashObjectCmd.String("t", "blob", "Type of object to hash")
		write := hashObjectCmd.Bool("w", false, "Write the object into object database")
		if err := hashObjectCmd.Parse(args[1:]); err != nil {
			return err
		}
		paths := hashObjectCmd.Args()
		if len(paths) == 0 {
			return fmt.Errorf("Expected path")
		}
		for _, path := range paths {
			err := repo.HashObject(*write, *objKind, path)
			if err != nil {
				return err
			}
		}
		return nil

	case "ls-tree":
		lsTreeCmd := flag.NewFlagSet("ls-tree", flag.ExitOnError)
		recursive := lsTreeCmd.Bool("r", false, "Recurse into sub-trees")
		if err := lsTreeCmd.Parse(args[1:]); err != nil {
			return err
		}
		treeish := lsTreeCmd.Args()
		if len(treeish) == 0 {
			return fmt.Errorf("Expected tree ref")
		}

		return repo.LsTree(treeish[0], *recursive)

	case "log":
		return repo.Log()

	case "show-ref":
		kind := ""
		if len(args) > 1 {
			kind = args[1][2:]
		}
		return repo.ShowRef(kind)

	default:
		return fmt.Errorf("%s command wasn't found\n", cmd)
	}
}

func (repo *Repository) isRef(sha string) string {
	ref := repo.refStore.heads[sha]
	return ref
}
