// +build !windows

package fs

import "os"

func (fs *FS) Chmod(path string, mode os.FileMode) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	return file.Chmod(mode)
}
