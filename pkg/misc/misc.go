package misc

import (
	"fmt"
	"os"
)

func DirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("path exists but is not a directory")
	}
	return true, nil
}

func MkdirIfNotExist(path string) error {
	exists, err := DirExists(path)
	if err != nil {
		return fmt.Errorf("unable to check if data dir exists: %w", err)
	}

	if !exists {
		return os.MkdirAll(path, 0755)
	}
	return nil
}
