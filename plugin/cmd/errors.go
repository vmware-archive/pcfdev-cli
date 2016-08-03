package cmd

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

type OldVMError struct{}

func (e *OldVMError) Error() string {
	return "old version of PCF Dev already running, please run `cf dev destroy` to continue"
}
