package repository

import (
	"fmt"
	"os"

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

func (repo *Repository) CatFile(objKind string, hash string) error {
	sha, err := repo.findObject(hash)
	if err != nil {
		return err
	}

	obj, err := repo.makeObject(sha)
	if err != nil {
		return err
	}

	switch objKind {
	case "blob":
		if blob, ok := obj.(*Blob); ok {
			fmt.Print(string(blob.contents))
		} else {
			return fmt.Errorf("Mismatched types. Object %s is not a blob", hash)
		}
	case "commit", "tree", "tag":
		fmt.Print(string(obj.Serialize()))
	default:
		return fmt.Errorf("Unknown object type :%s", objKind)
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
