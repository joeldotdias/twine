package helpers

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

func SearchRoot(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	if isDir(filepath.Join(absPath, ".git")) {
		return absPath, nil
	}

	parent := filepath.Dir(absPath)
	if parent == absPath {
		return "", fmt.Errorf("Not in a repository. Run twine init to make one.")
	}

	return SearchRoot(parent)
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}

func IsHex(str string) bool {
	_, err := hex.DecodeString(str)
	return err == nil
}
