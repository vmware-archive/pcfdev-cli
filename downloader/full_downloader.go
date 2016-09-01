package downloader

import (
	"errors"

	"github.com/pivotal-cf/pcfdev-cli/config"
)

type FullDownloader struct {
	Downloader OVADownloader
	FS         FS
	Config     *config.Config
}

func (f *FullDownloader) IsOVACurrent() (bool, error) {
	return f.Downloader.IsOVACurrent()
}

func (f *FullDownloader) Download() error {
	if err := f.Downloader.Setup(); err != nil {
		return err
	}

	md5, err := f.Downloader.Download()
	if err != nil {
		return err
	}

	if md5 != f.Config.ExpectedMD5 {
		return errors.New("download failed")
	}

	return f.FS.Move(f.Config.PartialOVAPath, f.Config.OVAPath)
}
