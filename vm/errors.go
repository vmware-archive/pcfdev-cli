package vm

import "fmt"

type StartVMError struct {
	Err error
}

func (e *StartVMError) Error() string {
	return fmt.Sprintf("failed to start VM: %s", e.Err)
}

type SuspendVMError struct {
	Err error
}

func (e *SuspendVMError) Error() string {
	return fmt.Sprintf("failed to suspend VM: %s", e.Err)
}

type ResumeVMError struct {
	Err error
}

func (e *ResumeVMError) Error() string {
	return fmt.Sprintf("failed to resume VM: %s", e.Err)
}

type ImportVMError struct {
	Err error
}

func (e *ImportVMError) Error() string {
	return fmt.Sprintf("failed to import VM: %s", e.Err)
}

type ProvisionVMError struct {
	Err error
}

func (e *ProvisionVMError) Error() string {
	return fmt.Sprintf("failed to provision VM: %s", e.Err)
}

type StopVMError struct {
	Err error
}

func (e *StopVMError) Error() string {
	return fmt.Sprintf("failed to stop VM: %s", e.Err)
}

type DestroyVMError struct {
	Err error
}

func (e *DestroyVMError) Error() string {
	return fmt.Sprintf("failed to destroy VM: %s", e.Err)
}
