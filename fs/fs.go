package fs

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

type FS struct{}

func (fs *FS) Exists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (fs *FS) Write(path string, contents io.ReadCloser) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %s", err)
	}
	defer file.Close()

	if _, err = io.Copy(file, contents); err != nil {
		return fmt.Errorf("failed to copy contents to file: %s", err)
	}
	return nil
}

func (fs *FS) CreateDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func (fs *FS) RemoveFile(path string) error {
	return os.Remove(path)
}

func (fs *FS) MD5(path string) (string, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("could not read %s: %s", path, err)
	}

	return fmt.Sprintf("%x", md5.Sum(contents)), nil
}
