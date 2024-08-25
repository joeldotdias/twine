package repository

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joeldotdias/twine/pkg/iniparse"
)

type Config struct {
	username      string
	email         string
	defaultBranch string
}

func makeCfg() Config {
	homedir, _ := os.UserHomeDir()
	cfg, err := iniparse.Read(filepath.Join(homedir, ".gitconfig"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't read global config file: %v", err)
	}

	init := cfg.Section("init")
	user := cfg.Section("user")

	return Config{
		username:      user.Key("name"),
		email:         user.Key("email"),
		defaultBranch: init.Key("defaultBranch"),
	}
}
