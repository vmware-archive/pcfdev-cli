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
	DeleteAllExcept(path string, filenames []string) error
}

//go:generate mockgen -package mocks -destination mocks/config.go github.com/pivotal-cf/pcfdev-cli/downloader Config
type Config interface {
	GetOVAPath() (ovaPath string, err error)
	SaveToken() error
}
type Downloader struct {
	FS           FS
	PivnetClient Client
	Config       Config
	ExpectedMD5  string
}

func (d *Downloader) partialFilePath(path string) string {
	return path + ".partial"
}

func (d *Downloader) IsOVACurrent() (bool, error) {
	path, err := d.Config.GetOVAPath()
	if err != nil {
		return false, err
	}

	fileExists, err := d.FS.Exists(path)
	if err != nil {
		return false, err
	}
	if !fileExists {
		return false, nil
	}

	md5, err := d.FS.MD5(path)
	if err != nil {
		return false, err
	}
	if md5 != d.ExpectedMD5 {
		return false, nil
	}

	return true, nil
}

func (d *Downloader) Download() error {
	path, err := d.Config.GetOVAPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	filename := filepath.Base(path)
	if err := d.FS.CreateDir(dir); err != nil {
		return err
	}

	if err := d.FS.DeleteAllExcept(dir, []string{filename, d.partialFilePath(filename)}); err != nil {
		return err
	}

	partialFileExists, err := d.FS.Exists(d.partialFilePath(path))
	if err != nil {
		return err
	}

	var md5 string
	if partialFileExists {
		md5, err = d.resumeDownload(path)
	} else {
		md5, err = d.download(path, 0)
	}
	if err != nil {
		return err
	}

	if md5 != d.ExpectedMD5 {
		return errors.New("download failed")
	}

	if err := d.FS.Move(d.partialFilePath(path), path); err != nil {
		return err
	}

	return nil
}

func (d *Downloader) resumeDownload(path string) (md5 string, err error) {
	startAtByte, err := d.FS.Length(d.partialFilePath(path))
	if err != nil {
		return "", err
	}

	md5, err = d.download(path, startAtByte)
	if md5 != d.ExpectedMD5 {
		if err := d.FS.RemoveFile(d.partialFilePath(path)); err != nil {
			return "", err
		}
		return d.download(path, 0)
	}
	return md5, nil
}

func (d *Downloader) download(path string, startAtBytes int64) (md5 string, err error) {
	ova, err := d.PivnetClient.DownloadOVA(startAtBytes)
	if err != nil {
		return "", err
	}
	defer ova.Close()

	if err := d.Config.SaveToken(); err != nil {
		return "", err
	}

	if err := d.FS.Write(d.partialFilePath(path), ova); err != nil {
		return "", err
	}

	return d.FS.MD5(d.partialFilePath(path))
}
