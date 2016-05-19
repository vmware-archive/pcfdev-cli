package user

import "os/user"

func IsPrivileged() (bool, error) {
	currentUser, err := user.Current()
	if err != nil {
		return false, err
	}

	return currentUser.Uid == "0", nil
}
