package user

import usr "os/user"

type User struct{}

func GetHome() (string, error) {
	u, err := usr.Current()
	if err != nil {
		return "", err
	}

	return u.HomeDir, nil
}
