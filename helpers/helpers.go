package helpers

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	gouuid "github.com/nu7hatch/gouuid"
)

func ImportSnappy() (vmName string, err error) {
	_, err = os.Stat("../assets/snappy.ova")
	if os.IsNotExist(err) {
		resp, err := http.Get("https://s3.amazonaws.com/pcfdev/ovas/snappy.ova")
		if err != nil {
			return "", err
		}

		ovaFile, err := os.Create("../assets/snappy.ova")
		if err != nil {
			return "", err
		}

		defer ovaFile.Close()
		_, err = io.Copy(ovaFile, resp.Body)
		if err != nil {
			return "", err
		}
	}

	tmpDir := os.Getenv("TMPDIR")
	uuid, err := gouuid.NewV4()
	if err != nil {
		return "", err
	}
	vmName = "Snappy-" + uuid.String()

	command := exec.Command("VBoxManage",
		"import",
		"../assets/snappy.ova",
		"--vsys", "0",
		"--vmname", vmName,
		"--unit", "6", "--disk", filepath.Join(tmpDir, vmName+"-disk1_4.vmdk"),
		"--unit", "7", "--disk", filepath.Join(tmpDir, vmName+"-disk2.vmdk"))
	err = command.Run()
	return vmName, err
}
