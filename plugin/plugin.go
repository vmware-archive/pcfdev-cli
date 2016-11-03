package plugin

import (
	"fmt"
	"strconv"
	"strings"

	cfplugin "github.com/cloudfoundry/cli/plugin"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/plugin/cmd"
)

type Plugin struct {
	UI         UI
	CmdBuilder CmdBuilder
	Exit       Exit
	Config     *config.Config
}

//go:generate mockgen -package mocks -destination mocks/ui.go github.com/pivotal-cf/pcfdev-cli/plugin UI
type UI interface {
	Failed(message string, args ...interface{})
	Say(message string, args ...interface{})
	Ask(prompt string) (answer string)
}

//go:generate mockgen -package mocks -destination mocks/cmd_builder.go github.com/pivotal-cf/pcfdev-cli/plugin CmdBuilder
type CmdBuilder interface {
	Cmd(subcommand string) (cmd.Cmd, error)
}

//go:generate mockgen -package mocks -destination mocks/exit.go github.com/pivotal-cf/pcfdev-cli/plugin Exit
type Exit interface {
	Exit()
}

//go:generate mockgen -package mocks -destination mocks/cmd.go github.com/pivotal-cf/pcfdev-cli/plugin/cmd Cmd

func (p *Plugin) Run(cliConnection cfplugin.CliConnection, args []string) {
	if args[0] == "CLI-MESSAGE-UNINSTALL" {
		return
	}

	var subcommand string
	var cmdArgs []string

	if len(args) > 1 {
		subcommand = args[1]
		cmdArgs = args[2:]
	}

	cmd, err := p.CmdBuilder.Cmd(subcommand)
	if err != nil {
		p.showUsageMessage(cliConnection)
		return
	}
	if cmd.Parse(cmdArgs) != nil {
		p.showUsageMessage(cliConnection)
		return
	}
	if err := cmd.Run(); err != nil {
		p.UI.Failed(getErrorText(err))
		p.Exit.Exit()
	}
}

func (p *Plugin) showUsageMessage(cliConnection cfplugin.CliConnection) {
	if _, err := cliConnection.CliCommand("help", "dev"); err != nil {
		p.UI.Failed(getErrorText(err))
		p.Exit.Exit()
	}
}

func getErrorText(err error) string {
	return fmt.Sprintf("Error: %s.", err.Error())
}

func (p *Plugin) getPluginVersion() cfplugin.VersionType {
	var majorVersion, minorVersion, buildVersion int
	var errMajor, errMinor, errBuild error

	versionParts := strings.SplitN(p.Config.Version.BuildVersion, ".", 3)

	if len(versionParts) == 3 {
		majorVersion, errMajor = strconv.Atoi(versionParts[0])
		minorVersion, errMinor = strconv.Atoi(versionParts[1])
		buildVersion, errBuild = strconv.Atoi(versionParts[2])
		if errMajor != nil || errMinor != nil || errBuild != nil {
			return cfplugin.VersionType{}
		}
	}

	return cfplugin.VersionType{
		Major: majorVersion,
		Minor: minorVersion,
		Build: buildVersion,
	}
}

func (p *Plugin) GetMetadata() cfplugin.PluginMetadata {
	return cfplugin.PluginMetadata{
		Name:    "pcfdev",
		Version: p.getPluginVersion(),
		Commands: []cfplugin.Command{
			cfplugin.Command{
				Name:     "dev",
				Alias:    "pcfdev",
				HelpText: "Control PCF Dev VMs running on your workstation",
				UsageDetails: cfplugin.Usage{
					Usage: `cf dev SUBCOMMAND

SUBCOMMANDS:
   start                             Start the PCF Dev VM. When creating a VM, http proxy env vars are respected.
      [-c number-of-cores]           Number of processor cores used by VM. Default: number of physical cores.
      [-d domain]                    Specify the domain that the PCF Dev VM will occupy.
      [-i ip-address]                Specify the IP Address that the PCF Dev VM will occupy.
      [-k]                           Import VM certificates into host's trusted certificate store.
      [-m memory-in-mb]              Memory to allocate for VM. Default: half of total memory, max 4 GB, max 8 GB with SCS.
      [-r registry1,registry2,...]   Docker registries that PCF Dev will use without SSL validation. Specify in 'host:port' format.
      [-s service1,service2]         Specify the services started with PCF Dev.
                                        Options: redis, rabbitmq, spring-cloud-services (scs), default, all, none
                                        Default: redis, rabbitmq
                                        (MySQL is always available and cannot be disabled.)
      [-t]                           Perform a CF login to PCF Dev after starting, as the 'user' user.
   stop                              Shutdown the PCF Dev VM. All data is preserved.
   suspend                           Save the current state of the PCF Dev VM to disk and then stop the VM.
   resume                            Resume PCF Dev VM from suspended state.
   destroy                           Delete the PCF Dev VM. All data is destroyed.
   status                            Query for the status of the PCF Dev VM.
   import /path/to/ova               Import OVA from local filesystem.
   ssh                               Start an SSH session into a running PCF Dev VM.
   target                            Perform a CF login to PCF Dev, as the 'user' user.
   trust                             Import VM certificates into host's trusted certificate store.
      [-p]                           Print the PCF Dev Root CA Certificate to stdout.
   untrust                           Remove VM certificates from host's trusted certificate store.
   version                           Display the release version of the CLI.`,
				},
			},
		},
	}
}
