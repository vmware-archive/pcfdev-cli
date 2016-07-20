package fs

import (
	"archive/tar"
	cMD5 "crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type FS struct{}

func (fs *FS) Exists(path string) (exists bool, err error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (fs *FS) Read(path string) (contents []byte, err error) {
	return ioutil.ReadFile(path)
}

func (fs *FS) Write(path string, contents io.Reader) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %s", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, contents); err != nil {
		return fmt.Errorf("failed to copy contents to file: %s", err)
	}
	return nil
}

func (fs *FS) CreateDir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %s", path, err)
	}

	return nil
}

func (fs *FS) DeleteAllExcept(path string, filenames []string) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to list files: %s", err)
	}

	for _, file := range files {
		if !fs.fileInSet(file.Name(), filenames) {
			err := fs.Remove(filepath.Join(path, file.Name()))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (fs *FS) Remove(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("failed to remove file %s: %s", path, err)
	}

	return nil
}

func (fs *FS) MD5(path string) (md5 string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open %s: %s", path, err)
	}
	defer file.Close()

	hash := cMD5.New()

	if _, err = io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to read %s: %s", path, err)
	}

	return fmt.Sprintf("%x", hash.Sum([]byte{})), nil
}

func (fs *FS) Length(path string) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("failed to read %s: %s", path, err)
	}
	defer file.Close()

	fileInfo, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	return fileInfo.Size(), nil
}

func (fs *FS) Move(source string, destination string) error {
	if err := os.Rename(source, destination); err != nil {
		return fmt.Errorf("failed to move %s to %s: %s", source, destination, err)
	}

	return nil
}

func (fs *FS) Copy(source string, destination string) error {
	data, err := ioutil.ReadFile(source)
	if err != nil {
		return fmt.Errorf("failed to copy %s to %s: %s", source, destination, err)
	}

	os.Remove(destination)
	if err != nil {
		return fmt.Errorf("failed to copy %s to %s: %s", source, destination, err)
	}

	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to copy %s to %s: %s", source, destination, err)
	}
	defer file.Close()

	if _, err = io.Copy(file, ioutil.NopCloser(strings.NewReader(string(data)))); err != nil {
		return fmt.Errorf("failed to copy %s to %s: %s", source, destination, err)
	}

	return nil
}

func (fs *FS) Extract(archivePath string, destinationPath string, pattern string) error {
	archive, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open %s: %s", archivePath, err)
	}

	reader := tar.NewReader(archive)

	regex := regexp.MustCompile(pattern)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("malformed tar %s:%s", archivePath, err)
		}
		matches := regex.FindStringSubmatch(header.Name)
		if len(matches) > 0 {
			fs.Write(destinationPath, reader)
			return nil
		}
	}

	return fmt.Errorf("could not find file matching %s in %s", regex, archivePath)
}

func (fs *FS) fileInSet(filenameToFind string, filenames []string) bool {
	for _, filename := range filenames {
		if filenameToFind == filename {
			return true
		}
	}
	return false
}
