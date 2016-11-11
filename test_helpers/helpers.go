package test_helpers

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	guid "github.com/nu7hatch/gouuid"
	"github.com/pivotal-cf/pcfdev-cli/helpers"
)

func ImportSnappy() (vmName string, err error) {
	_, err = os.Stat(filepath.Join("..", "assets", "snappy.ova"))
	if os.IsNotExist(err) {
		resp, err := http.Get("https://s3.amazonaws.com/pcfdev/ovas/snappy.ova")
		if err != nil {
			return "", err
		}

		ovaFile, err := os.Create(filepath.Join("..", "assets", "snappy.ova"))
		if err != nil {
			return "", err
		}

		defer ovaFile.Close()
		_, err = io.Copy(ovaFile, resp.Body)
		if err != nil {
			return "", err
		}
	}

	tmpDir := os.TempDir()

	uuid, err := guid.NewV4()
	if err != nil {
		return "", err
	}
	vmName = "Snappy-" + uuid.String()

	vBoxManagePath, err := helpers.VBoxManagePath()
	if err != nil {
		return "", err
	}

	command := exec.Command(vBoxManagePath,
		"import",
		filepath.Join("..", "assets", "snappy.ova"),
		"--vsys", "0",
		"--vmname", vmName,
		"--unit", "6", "--disk", filepath.Join(tmpDir, vmName+"-disk1_4.vmdk"),
		"--unit", "7", "--disk", filepath.Join(tmpDir, vmName+"-disk2.vmdk"))
	err = command.Run()
	return vmName, err
}
