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
