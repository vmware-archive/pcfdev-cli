package plugin

import "strings"

type NonTranslatingUI struct {
	UI
}

func (ui *NonTranslatingUI) Confirm(message string) bool {
	response := ui.Ask(message)
	switch strings.ToLower(response) {
	case "y", "yes":
		return true
	}
	return false
}

func (ui *NonTranslatingUI) Failed(message string, args ...interface{}) {
	defer func() { recover() }()
	ui.UI.Failed(message, args...)
}
