package downloader

import (
	"errors"

	"github.com/pivotal-cf/pcfdev-cli/config"
)

type PartialDownloader struct {
	Downloader OVADownloader
	FS         FS
	Config     *config.Config
}

func (p *PartialDownloader) IsOVACurrent() (bool, error) {
	return p.Downloader.IsOVACurrent()
}

func (p *PartialDownloader) Download() error {
	if err := p.Downloader.Setup(); err != nil {
		return err
	}

	md5, err := p.Downloader.Download()
	if err != nil {
		return err
	}

	if md5 != p.Config.ExpectedMD5 {
		if err := p.FS.Remove(p.Config.PartialOVAPath); err != nil {
			return err
		}

		md5, err = p.Downloader.Download()
		if err != nil {
			return err
		}

		if md5 != p.Config.ExpectedMD5 {
			return errors.New("download failed")
		}

	}

	return p.FS.Move(p.Config.PartialOVAPath, p.Config.OVAPath)
}
