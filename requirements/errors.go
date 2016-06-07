package requirements

import "fmt"

type NotEnoughMemoryError struct {
	FreeMemory    uint64
	DesiredMemory uint64
}

func (e *NotEnoughMemoryError) Error() string {
	return ""
}

type RequestedMemoryTooLittleError struct {
	DesiredMemory uint64
	MinMemory     uint64
}

func (e *RequestedMemoryTooLittleError) Error() string {
	return fmt.Sprintf("PCF Dev requires at least %d MB of memory to run.", e.MinMemory)
}
