package downloader

import (
	"time"

	"github.com/pivotal-cf/pcfdev-cli/config"
)

//go:generate mockgen -package mocks -destination mocks/ova_downloader.go github.com/pivotal-cf/pcfdev-cli/downloader OVADownloader
type OVADownloader interface {
	Setup() error
	Download() (md5 string, err error)
	IsOVACurrent() (current bool, err error)
}

type Downloader interface {
	IsOVACurrent() (current bool, err error)
	Download() error
}

type DownloaderFactory struct {
	FS                   FS
	Config               *config.Config
	PivnetClient         Client
	Token                Token
	DownloadAttempts     int
	DownloadAttemptDelay time.Duration
}

func (f *DownloaderFactory) Create() (Downloader, error) {
	exists, err := f.FS.Exists(f.Config.PartialOVAPath)
	if err != nil {
		return nil, err
	}

	ovaDownloader := &ConcreteOVADownloader{
		FS:                   f.FS,
		Config:               f.Config,
		PivnetClient:         f.PivnetClient,
		Token:                f.Token,
		DownloadAttempts:     f.DownloadAttempts,
		DownloadAttemptDelay: f.DownloadAttemptDelay,
	}
	if exists {
		return &PartialDownloader{
			Downloader: ovaDownloader,
			FS:         f.FS,
			Config:     f.Config,
		}, nil
	} else {
		return &FullDownloader{
			Downloader: ovaDownloader,
			FS:         f.FS,
			Config:     f.Config,
		}, nil
	}
}
