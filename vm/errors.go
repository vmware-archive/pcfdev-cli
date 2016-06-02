package vm

import "fmt"

type StartVMError struct {
	error
}

func (e *StartVMError) Error() string {
	return fmt.Sprintf("failed to start VM: %s", e.error)
}

type SuspendVMError struct {
	error
}

func (e *SuspendVMError) Error() string {
	return fmt.Sprintf("failed to suspend VM: %s", e.error)
}

type ResumeVMError struct {
	error
}

func (e *ResumeVMError) Error() string {
	return fmt.Sprintf("failed to resume VM: %s", e.error)
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
	return fmt.Sprintf("failed to import VM: %s", e.error)
}

type ProvisionVMError struct {
	error
}

func (e *ProvisionVMError) Error() string {
	return fmt.Sprintf("failed to provision VM: %s", e.error)
}

type StopVMError struct {
	error
}

func (e *StopVMError) Error() string {
	return fmt.Sprintf("failed to stop VM: %s", e.error)
}

type DestroyVMError struct {
	error
}

func (e *DestroyVMError) Error() string {
	return fmt.Sprintf("failed to destroy VM: %s", e.error)
}
