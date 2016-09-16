package cert

import (
	"io"
	"path/filepath"
	"strings"
)

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/pivotal-cf/pcfdev-cli/cert FS
type FS interface {
	Exists(path string) (exists bool, err error)
	Read(path string) (contents []byte, err error)
	Remove(path string) error
	TempDir() (string, error)
	Write(path string, contents io.Reader) error
}

//go:generate mockgen -package mocks -destination mocks/system_store.go github.com/pivotal-cf/pcfdev-cli/cert SystemStore
type SystemStore interface {
	Store(path string) error
	Unstore() error
}

type CertStore struct {
	FS          FS
	SystemStore SystemStore
}

func (c *CertStore) Store(cert string) error {
	tempDir, err := c.FS.TempDir()
	if err != nil {
		return err
	}
	defer c.FS.Remove(tempDir)

	if err := c.FS.Write(filepath.Join(tempDir, "cert"), strings.NewReader(cert)); err != nil {
		return err
	}

	return c.SystemStore.Store(filepath.Join(tempDir, "cert"))
}

func (c *CertStore) Unstore() error {
	return c.SystemStore.Unstore()
}
