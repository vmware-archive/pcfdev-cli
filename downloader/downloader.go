package downloader

import (
	"errors"
	"io"
	"path/filepath"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/pivnet"
)

const DOWNLOAD_ATTEMPTS = 3

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
	Remove(path string) error
	DeleteAllExcept(path string, filenames []string) error
}

//go:generate mockgen -package mocks -destination mocks/token.go github.com/pivotal-cf/pcfdev-cli/downloader Token
type Token interface {
	Save() error
}

type Downloader struct {
	FS                   FS
	PivnetClient         Client
	Config               *config.Config
	Token                Token
	DownloadAttempts     int
	DownloadAttemptDelay time.Duration
}

func (d *Downloader) partialFile(path string) string {
	return path + ".partial"
}

func (d *Downloader) IsOVACurrent() (bool, error) {
	path := filepath.Join(d.Config.OVADir, d.Config.DefaultVMName+".ova")

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
	if md5 != d.Config.ExpectedMD5 {
		return false, nil
	}

	return true, nil
}

func (d *Downloader) Download() error {
	dir := d.Config.OVADir
	filename := d.Config.DefaultVMName + ".ova"
	path := filepath.Join(dir, filename)
	partial := d.partialFile(filename)
	partialPath := d.partialFile(path)

	if err := d.FS.CreateDir(dir); err != nil {
		return err
	}

	if err := d.FS.DeleteAllExcept(dir, []string{filename, partial}); err != nil {
		return err
	}

	partialFileExists, err := d.FS.Exists(partialPath)
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

	if md5 != d.Config.ExpectedMD5 {
		return errors.New("download failed")
	}

	if err := d.FS.Move(partialPath, path); err != nil {
		return err
	}

	return nil
}

func (d *Downloader) resumeDownload(path string) (md5 string, err error) {
	startAtByte, err := d.FS.Length(d.partialFile(path))
	if err != nil {
		return "", err
	}

	md5, err = d.download(path, startAtByte)
	if md5 != d.Config.ExpectedMD5 {
		if err := d.FS.Remove(d.partialFile(path)); err != nil {
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

	if err := d.Token.Save(); err != nil {
		return "", err
	}

	for attempts := 0; attempts < d.DownloadAttempts; attempts++ {
		if err = d.FS.Write(d.partialFile(path), ova); err == nil {
			break
		}
		time.Sleep(d.DownloadAttemptDelay)
	}
	if err != nil {
		return "", err
	}

	return d.FS.MD5(d.partialFile(path))
}
