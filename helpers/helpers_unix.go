// +build !windows

package helpers

func VBoxManagePath() (path string, err error) {
	return "VBoxManage", nil
}
