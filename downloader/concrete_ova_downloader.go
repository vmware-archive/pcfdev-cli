package downloader

import (
	"io"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/helpers"
	"github.com/pivotal-cf/pcfdev-cli/pivnet"
)

type ConcreteOVADownloader struct {
	FS                   FS
	PivnetClient         Client
	Config               *config.Config
	Token                Token
	DownloadAttempts     int
	DownloadAttemptDelay time.Duration
}

//go:generate mockgen -package mocks -destination mocks/client.go github.com/pivotal-cf/pcfdev-cli/downloader Client
type Client interface {
	DownloadOVA(startAtByte int64) (ova *pivnet.DownloadReader, err error)
}

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/pivotal-cf/pcfdev-cli/downloader FS
type FS interface {
	Remove(path string) error
	Exists(path string) (exists bool, err error)
	MD5(path string) (md5 string, err error)
	CreateDir(path string) error
	Length(path string) (bytes int64, err error)
	Write(path string, contents io.Reader) error
	Move(source string, destinationPath string) error
	DeleteAllExcept(path string, filenames []string) error
}

//go:generate mockgen -package mocks -destination mocks/token.go github.com/pivotal-cf/pcfdev-cli/downloader Token
type Token interface {
	Save() error
}

func (d *ConcreteOVADownloader) IsOVACurrent() (bool, error) {
	fileExists, err := d.FS.Exists(d.Config.OVAPath)
	if err != nil {
		return false, err
	}
	if !fileExists {
		return false, nil
	}

	md5, err := d.FS.MD5(d.Config.OVAPath)
	if err != nil {
		return false, err
	}
	if md5 != d.Config.ExpectedMD5 {
		return false, nil
	}

	return true, nil
}

func (d *ConcreteOVADownloader) Setup() error {
	if err := d.FS.CreateDir(d.Config.OVADir); err != nil {
		return err
	}

	return d.FS.DeleteAllExcept(d.Config.OVADir, []string{d.Config.DefaultVMName + ".ova", d.Config.DefaultVMName + ".ova.partial"})
}

func (d *ConcreteOVADownloader) Download() (string, error) {
	err := helpers.ExecuteWithAttempts(func() error {
		exists, err := d.FS.Exists(d.Config.PartialOVAPath)
		if err != nil {
			return err
		}

		var startAtBytes int64
		if exists {
			startAtBytes, err = d.FS.Length(d.Config.PartialOVAPath)
			if err != nil {
				return err
			}
		} else {
			startAtBytes = int64(0)
		}

		ova, err := d.PivnetClient.DownloadOVA(startAtBytes)
		if err != nil {
			return err
		}
		defer ova.Close()

		if err := d.Token.Save(); err != nil {
			return err
		}

		if err := d.FS.Write(d.Config.PartialOVAPath, ova); err != nil {
			return err
		}

		return nil
	}, d.DownloadAttempts, d.DownloadAttemptDelay)

	if err != nil {
		return "", err
	}

	return d.FS.MD5(d.Config.PartialOVAPath)
}
