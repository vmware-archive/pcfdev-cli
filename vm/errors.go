package vm

import "fmt"

type StartVMError struct {
	error
}

func (e *StartVMError) Error() string {
	return fmt.Sprintf("could not start PCF Dev: %s", e.error)
}

type OldVMError struct {
}

func (e *OldVMError) Error() string {
	return "old version of PCF Dev already running"
}

type ImportVMError struct {
	error
}

func (e *ImportVMError) Error() string {
	return fmt.Sprintf("failed to import vm: %s", e.error)
}

type ProvisionVMError struct {
	error
}

func (e *ProvisionVMError) Error() string {
	return fmt.Sprintf("failed to provision vm: %s", e.error)
}

type StopVMError struct {
	error
}

func (e *StopVMError) Error() string {
	return fmt.Sprintf("failed to stop vm: %s", e.error)
}

type DestroyVMError struct {
	error
}

func (e *DestroyVMError) Error() string {
	return fmt.Sprintf("failed to destroy vm: %s", e.error)
}
