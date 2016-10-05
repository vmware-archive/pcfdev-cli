package pivnet

import "fmt"

type InvalidTokenError struct{}

func (e *InvalidTokenError) Error() string {
	return "invalid Pivotal Network API token"
}

type InvalidUsernameError struct{}

func (e *InvalidUsernameError) Error() string {
	return "unable to locate Pivotal Network user"
}

type InvalidPasswordError struct{}

func (e *InvalidPasswordError) Error() string {
	return "invalid password"
}

type UnexpectedResponseError struct {
	Err error
}

func (e *UnexpectedResponseError) Error() string {
	return fmt.Sprintf("%s", e.Err)
}

type PivNetUnreachableError struct {
	Err error
}

func (e *PivNetUnreachableError) Error() string {
	return fmt.Sprintf("failed to reach Pivotal Network: %s", e.Err)
}

type JSONUnmarshalError struct {
	Err error
}

func (e *JSONUnmarshalError) Error() string {
	return fmt.Sprintf("failed to parse network response: %s", e.Err)
}
