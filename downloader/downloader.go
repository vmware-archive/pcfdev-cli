package downloader

import (
	"errors"
	"io"
	"path/filepath"

	"github.com/pivotal-cf/pcfdev-cli/pivnet"
)

//go:generate mockgen -package mocks -destination mocks/client.go github.com/pivotal-cf/pcfdev-cli/downloader Client
type Client interface {
	DownloadOVA(startAtByte int64) (ova *pivnet.DownloadReader, err error)
}

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/pivotal-cf/pcfdev-cli/downloader FS
type FS interface {
	Exists(path string) (exists bool, err error)
	CreateDir(path string) error
	Length(path string) (bytes int64, err error)
	Write(path string, contents io.Reader) error
	MD5(path string) (md5 string, err error)
	Move(source string, destinationPath string) error
	RemoveFile(path string) error
}

type Downloader struct {
	FS           FS
	PivnetClient Client
	ExpectedMD5  string
}

func (d *Downloader) partialFilePath(path string) string {
	return path + ".partial"
}

func (d *Downloader) Download(path string) error {
	if err := d.FS.CreateDir(filepath.Dir(path)); err != nil {
		return err
	}

	fileExists, err := d.FS.Exists(path)
	if err != nil {
		return err
	}
	if fileExists {
		return nil
	}

	partialFileExists, err := d.FS.Exists(d.partialFilePath(path))
	if err != nil {
		return err
	}

	var startAtByte int64
	if partialFileExists {
		startAtByte, err = d.FS.Length(d.partialFilePath(path))
		if err != nil {
			return err
		}
	}

	md5, err := d.download(path, startAtByte)
	if err != nil {
		return err
	}
	if md5 != d.ExpectedMD5 {
		if partialFileExists {
			if err := d.FS.RemoveFile(d.partialFilePath(path)); err != nil {
				return err
			}

			md5, err = d.download(path, 0)
			if err != nil {
				return err
			}
			if md5 != d.ExpectedMD5 {
				return errors.New("download failed")
			}
		} else {
			return errors.New("download failed")
		}
	}

	if err := d.FS.Move(d.partialFilePath(path), path); err != nil {
		return err
	}

	return nil
}

func (d *Downloader) download(path string, startAtBytes int64) (md5 string, err error) {
	ova, err := d.PivnetClient.DownloadOVA(startAtBytes)
	if err != nil {
		return "", err
	}
	defer ova.Close()

	if err := d.FS.Write(d.partialFilePath(path), ova); err != nil {
		return "", err
	}

	return d.FS.MD5(d.partialFilePath(path))
}
