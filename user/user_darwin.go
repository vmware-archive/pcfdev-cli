package user

import "os/user"

type User struct{}

func (*User) IsPrivileged() (bool, error) {
	currentUser, err := user.Current()
	if err != nil {
		return false, err
	}

	return currentUser.Uid == "0", nil
}
