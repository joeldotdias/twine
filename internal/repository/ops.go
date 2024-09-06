package repository

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/joeldotdias/twine/pkg/iniparse"
)

func (repo *Repository) init() error {
	dirs := map[string][]string{
		"objects":  {"info", "pack"},
		"refs":     {"heads", "tags"},
		"info":     {},
		"hooks":    {},
		"branches": {},
	}

	for dir, subdirs := range dirs {
		if len(subdirs) == 0 {
			path := repo.makePath(dir)
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
		}

		for _, subsubdir := range subdirs {
			subpath := repo.makePath(dir, subsubdir)
			if err := os.MkdirAll(subpath, 0o755); err != nil {
				return err
			}

		}
	}

	toWrite := map[string]string{
		"HEAD":        "ref: refs/heads/" + repo.conf.defaultBranch + "\n",
		"description": "Unnamed repository; edit this file 'description' to name the repository.\n",
		"info/exclude": "# git ls-files --others --exclude-from=.git/info/exclude\n" +
			"# Lines that start with '#' are comments.\n" +
			"# For a project mostly in C, the following would be a good set of\n" +
			"# exclude patterns (uncomment them if you want to use them):\n" +
			"# *.[oa]\n" +
			"# *~\n",
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

func (repo *Repository) catFile(args []string) error {
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

func (repo *Repository) hashObject(write bool, objKind string, path string) error {
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

func (repo *Repository) lsTree(treeish string, recursive bool) error {
	return repo.walkTree(treeish, recursive, "")
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
		return fmt.Errorf("Object %s is neither a tree nor a commit. It's a %s", sha, obj.Kind())
	}

	for _, leaf := range tree.leaves {
		var typeStr string
		shaStr := hex.EncodeToString(leaf.sha)

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
				shaStr,
				filepath.Join(prefix, leaf.path))
		}

		if recursive && typeStr == "tree" {
			err := repo.walkTree(shaStr, recursive, filepath.Join(prefix, leaf.path))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (repo *Repository) log() error {
	_, err := repo.makeCommitLog("HEAD")
	if err != nil {
		return fmt.Errorf("Couldn't parse commit log: %s", err)
	}

	return nil
}

func (repo *Repository) makeCommitLog(ref string) (*Commit, error) {
	sha, err := repo.findObject(ref)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		return nil, err
	}
	obj, err := repo.makeObject(sha)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		return nil, err
	}

	commit, ok := obj.(*Commit)
	if !ok {
		return nil, nil
	}

	commitLog, parent, err := commit.parseCommitLog(sha, func() (string, bool) {
		return repo.isRef(sha), ref == "HEAD"
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't parse commit log: %s", err)
	}

	fmt.Print(commitLog)

	if len(parent) > 0 {
		commit, err = repo.makeCommitLog(parent)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
		}
	}

	return commit, nil
}

func (repo *Repository) showRef(kind string) error {
	fmt.Println(kind)
	var headRefs, tagRefs, showRefs []string
	extractRef := func(ref string) string {
		return strings.SplitN(strings.SplitN(ref, " ", 2)[1], "/", 3)[2]
	}

	for sha, refName := range repo.refStore.heads {
		headRefs = append(headRefs, fmt.Sprintf("%s refs/heads/%s", sha, refName))
	}
	sort.Slice(headRefs, func(i, j int) bool {
		return extractRef(headRefs[i]) < extractRef(headRefs[j])
	})

	for sha, refName := range repo.refStore.tags {
		tagRefs = append(tagRefs, fmt.Sprintf("%s refs/tags/%s", sha, refName))
	}
	sort.Slice(tagRefs, func(i, j int) bool {
		return extractRef(tagRefs[i]) < extractRef(tagRefs[j])
	})

	switch kind {
	case "heads", "branches":
		showRefs = headRefs
	case "tags":
		showRefs = tagRefs
	default:
		showRefs = append(headRefs, tagRefs...)
	}

	for _, ref := range showRefs {
		fmt.Println(ref)
	}

	return nil
}

func (repo *Repository) listTags() error {
	tagsPath := repo.makePath("refs", "tags")
	files, err := os.ReadDir(tagsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return fmt.Errorf("Couldn't read tags: %s", err)
		}
	}

	for _, file := range files {
		fmt.Println(file.Name())
	}

	return nil
}

func (repo *Repository) createTag(args []string) error {
	var tagname, message string
	commit := "HEAD"
	alen := len(args)
	if alen == 1 {
		tagname = args[0]
		return repo.createLightweightTag(tagname, "HEAD")
	} else if alen == 2 {
		if args[0] == "-d" {
			tagname = args[1]
			return repo.deleteTag(tagname)
		} else {
			tagname = args[0]
			commit = args[1]
			return repo.createLightweightTag(tagname, commit)
		}
	} else if args[0] == "-a" {
		tagname = args[1]
		if args[2] != "-m" {
			commit = args[2]
			message = args[4]
		} else {
			message = args[3]
		}

		return repo.createAnnotatedTag(tagname, commit, message)
	}

	return nil
}

func (repo *Repository) createAnnotatedTag(name, ref, message string) error {
	sha, err := repo.findObject(ref)
	if err != nil {
		return fmt.Errorf("Couldn't find ref %s: %s", ref, err)
	}
	obj, err := repo.makeObject(sha)
	if err != nil {
		return fmt.Errorf("Couldn't find object %s: %s", sha, err)
	}

	_, ok := obj.(*Commit)
	if !ok {
		return fmt.Errorf("Tags can only be created on commits but %s is a %s", ref, obj.Kind())
	}

	tagger := fmt.Sprintf("%s <%s> %d +0000", repo.conf.username, repo.conf.email, time.Now().Unix())

	tag := &Tag{
		metaKV: map[TagField][]string{
			"object": {sha},
			"type":   {"commit"},
			"tag":    {name},
			"tagger": {tagger},
		},
		message: message,
	}

	tagSha, err := repo.writeObject(tag, true)
	if err != nil {
		return fmt.Errorf("Couldn't write tag object: %s", err)
	}

	return os.WriteFile(repo.makePath("refs", "tags", name), []byte(tagSha), 0o644)
}

func (repo *Repository) createLightweightTag(name, ref string) error {
	sha, _ := repo.findObject(ref)
	_, err := repo.makeObject(sha)
	if err != nil {
		return err
	}

	return os.WriteFile(repo.makePath("refs", "tags", name), []byte(sha), 0o644)
}

func (repo *Repository) deleteTag(name string) error {
	err := os.Remove(repo.makePath("refs", "tags", name))
	if err != nil {
		return fmt.Errorf("Couldn't delete tag %s: %w", name, err)
	}

	return nil
}

func (repo *Repository) lsFiles(args []string) error {
	if len(args) == 0 {
		for _, entry := range repo.index.entries {
			fmt.Println(entry.path)
		}
	}

	return nil
}
