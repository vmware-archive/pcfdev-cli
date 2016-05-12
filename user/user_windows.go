package user

type User struct{}

func (u *User) IsPrivileged() (bool, error) {
	return true, nil
}
