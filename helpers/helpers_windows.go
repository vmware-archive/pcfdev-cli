package helpers

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

func VBoxManagePath() (path string, err error) {
	vBoxManagePath, err := exec.LookPath("VBoxManage")
	if err != nil {
		if os.Getenv("VBOX_INSTALL_PATH") != "" {
			vBoxManagePath = filepath.Join(os.Getenv("VBOX_INSTALL_PATH"), "VBoxManage")
		}

		if os.Getenv("VBOX_MSI_INSTALL_PATH") != "" {
			vBoxManagePath = filepath.Join(os.Getenv("VBOX_MSI_INSTALL_PATH"), "VBoxManage")
		}

		if vBoxManagePath == "" {
			return "", errors.New("could not find VBoxManage executable")
		}
	}
	return vBoxManagePath, nil
}
