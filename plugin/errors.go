package plugin

import "fmt"

type EULARefusedError struct{}

func (e *EULARefusedError) Error() string {
	return "you must accept the end user license agreement to use PCF Dev"
}

type DestroyVMError struct {
	Err error
}

func (e *DestroyVMError) Error() string {
	return fmt.Sprintf("failed to destroy VM: %s", e.Err)
}
