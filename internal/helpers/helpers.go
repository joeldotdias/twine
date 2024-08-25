package helpers

import (
	"encoding/hex"
	"os"
)

func IsDir(path string) bool {
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
