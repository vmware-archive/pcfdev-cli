package pivnet

import "fmt"

type InvalidTokenError struct{}

func (e *InvalidTokenError) Error() string {
	return "invalid Pivotal Network API token"
}

type UnexpectedResponseError struct {
	error
}

func (e *UnexpectedResponseError) Error() string {
	return e.error.Error()
}

type PivNetUnreachableError struct {
	error
}

func (e *PivNetUnreachableError) Error() string {
	return fmt.Sprintf("failed to reach Pivotal Network: %s", e.error.Error())
}

type JSONUnmarshalError struct {
	error
}

func (e *JSONUnmarshalError) Error() string {
	return fmt.Sprintf("failed to parse network response: %s", e.error.Error())
}
