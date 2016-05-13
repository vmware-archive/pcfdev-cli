package user

type User struct{}

func (*User) IsPrivileged() (bool, error) {
	return true, nil
}
