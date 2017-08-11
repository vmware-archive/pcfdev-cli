package ui

import "fmt"

//go:generate mockgen -package mocks -destination mocks/plugin_ui.go github.com/pivotal-cf/pcfdev-cli/ui PluginUI
type PluginUI interface {
	Say(message string, args ...interface{})
}

type HelpText struct {
	UI PluginUI
}

func (h *HelpText) Print(domain string, autoTarget bool) {
	h.UI.Say(` _______  _______  _______    ______   _______  __   __
|       ||       ||       |  |      | |       ||  | |  |
|    _  ||       ||    ___|  |  _    ||    ___||  |_|  |
|   |_| ||       ||   |___   | | |   ||   |___ |       |
|    ___||      _||    ___|  | |_|   ||    ___||       |
|   |    |     |_ |   |      |       ||   |___  |     |
|___|    |_______||___|      |______| |_______|  |___|
is now running.`)

	if autoTarget {
		h.UI.Say(`PCF Dev automatically targeted. To target manually, run:`)
	} else {
		h.UI.Say(`To begin using PCF Dev, please run:`)
	}

	h.UI.Say(fmt.Sprintf(`   cf login -a https://api.%s --skip-ssl-validation
Apps Manager URL: https://apps.%s
Admin user => Email: admin / Password: admin
Regular user => Email: user / Password: pass`, domain, domain))
}
