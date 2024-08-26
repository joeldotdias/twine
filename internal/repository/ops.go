package repository

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joeldotdias/twine/pkg/iniparse"
)

func (repo *Repository) Init() error {
	for _, dir := range []string{repo.gitDir, repo.makePath("objects"), repo.makePath("refs"), repo.makePath("branches")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	toWrite := map[string]string{
		"HEAD":        "ref: refs/heads/" + repo.conf.defaultBranch + "\n",
		"description": "Unnamed repository; edit this file 'description' to name the repository.\n",
	}
	for fname, contents := range toWrite {
		if err := os.WriteFile(repo.makePath(fname), []byte(contents), 0o644); err != nil {
			return err
		}
	}

	configContents := iniparse.New()
	defaultConfig := map[string]string{
		"repositoryformatversion": "0",
		"filemode":                "true",
		"bare":                    "false",
	}
	coreSec := configContents.NewSection("core")
	for k, v := range defaultConfig {
		coreSec.NewKV(k, v)
	}

	err := configContents.Write(repo.makePath("config"))
	if err != nil {
		return fmt.Errorf("Couldn't write config file: %v\n", err)
	}

	fmt.Println("Initialized Twine repository")
	return nil
}

func (repo *Repository) CatFile(args []string) error {
	catFileCmd := flag.NewFlagSet("cat-file", flag.ExitOnError)

	typeFlag := catFileCmd.Bool("t", false, "Show object type")
	sizeFlag := catFileCmd.Bool("s", false, "Show object size")
	prettyFlag := catFileCmd.Bool("p", false, "Pretty-print object's content")

	if err := catFileCmd.Parse(args); err != nil {
		return err
	}

	valArgs := catFileCmd.Args()
	if len(valArgs) == 0 || len(valArgs) > 2 {
		return fmt.Errorf("You must provide exactly one object hash")
	}

	var objKind, hash string
	if len(valArgs) == 2 {
		objKind = valArgs[0]
		hash = valArgs[1]
	} else {
		hash = valArgs[0]
	}

	sha, err := repo.findObject(hash)
	if err != nil {
		return err
	}

	obj, err := repo.makeObject(sha)
	if err != nil {
		return err
	}

	switch {
	case *typeFlag:
		fmt.Println(obj.Kind())

	case *sizeFlag:
		fmt.Println(len(obj.Serialize()))

	case *prettyFlag || objKind != "":
		if objKind != "" && objKind != obj.Kind() {
			return fmt.Errorf("object %s is a %s, not a %s", hash, obj.Kind(), objKind)
		}

		fmt.Print(string(obj.Serialize()))

	default:
		return fmt.Errorf("Expected an option (-t, -s, -p) or an object type")
	}

	return nil
}

func (repo *Repository) HashObject(write bool, objKind string, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("Couldn't open file %v: %s", file, err)
	}
	defer file.Close()

	sha, err := repo.makeObjectHash(file, objKind, write)
	if err != nil {
		return fmt.Errorf("Couldn't hash object: %v", err)
	}

	fmt.Println(sha)

	return nil
}

func (repo *Repository) LsTree(tree string, recursive bool) error {
	return repo.walkTree(tree, recursive, "")
}

func (repo *Repository) walkTree(ref string, recursive bool, prefix string) error {
	sha, _ := repo.findObject(ref)
	obj, err := repo.makeObject(sha)
	if err != nil {
		return err
	}

	var tree *Tree
	switch o := obj.(type) {
	case *Commit:
		treeSha, err := o.getField("tree")
		if err != nil {
			return err
		}
		treeObj, err := repo.makeObject(treeSha)
		if err != nil {
			return err
		}

		var ok bool
		tree, ok = treeObj.(*Tree)
		if !ok {
			return fmt.Errorf("Couldn't get tree object from this commit")
		}

	case *Tree:
		tree = o

	default:
		return fmt.Errorf("Object %s is neither a tree nor a commit. It's type is %s", sha, obj.Kind())
	}

	for _, leaf := range tree.leaves {
		var typeStr string

		switch leaf.mode {
		case "40000":
			typeStr = "tree"
		case "100644", "100664", "100755":
			typeStr = "blob"
		case "120000":
			typeStr = "blob" // but it's a symlink
		case "160000":
			typeStr = "commit"
		default:
			return fmt.Errorf("unknown mode %s", leaf.mode)
		}

		if !recursive || (recursive && typeStr == "blob") {
			fmt.Printf("%06s %s %s\t%s\n",
				leaf.mode,
				typeStr,
				hex.EncodeToString(leaf.sha),
				filepath.Join(prefix, leaf.path))
		}

		if recursive && typeStr == "tree" {
			err := repo.walkTree(hex.EncodeToString(leaf.sha), recursive, filepath.Join(prefix, leaf.path))
			if err != nil {
				return err
			}
		}
	}

	return nil
}
