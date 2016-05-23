package plugin

import "fmt"

type EULARefusedError struct{}

func (e *EULARefusedError) Error() string {
	return "the user did not accept the eula"
}

type StartError struct {
	error
}

func (e *StartError) Error() string {
	return fmt.Sprintf("could not start PCF Dev: %s", e.Error())
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
	return fmt.Sprintf("failed to import vm: %s", e.Error())
}

type ProvisionVMError struct {
	error
}

func (e *ProvisionVMError) Error() string {
	return fmt.Sprintf("failed to provision vm: %s", e.Error())
}

type StopVMError struct {
	error
}

func (e *StopVMError) Error() string {
	return fmt.Sprintf("failed to stop vm: %s", e.Error())
}

type DestroyVMError struct {
	error
}

func (e *DestroyVMError) Error() string {
	return fmt.Sprintf("failed to destroy vm: %s", e.Error())
}
