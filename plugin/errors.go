package plugin

type EULARefusedError struct{}

func (e *EULARefusedError) Error() string {
	return "the user did not accept the eula"
}
