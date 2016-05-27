package plugin

import "fmt"

type EULARefusedError struct{}

func (e *EULARefusedError) Error() string {
	return "the user did not accept the eula"
}

type DestroyVMError struct {
	error
}

func (e *DestroyVMError) Error() string {
	return fmt.Sprintf("failed to destroy vm: %s", e.Error())
}
