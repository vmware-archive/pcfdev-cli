package plugin_test

import (
	"errors"

	cfplugin "github.com/cloudfoundry/cli/plugin"
	"github.com/cloudfoundry/cli/plugin/pluginfakes"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/plugin"
	"github.com/pivotal-cf/pcfdev-cli/plugin/mocks"
	"github.com/pivotal-cf/pcfdev-cli/user"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plugin", func() {
	var (
		mockCtrl          *gomock.Controller
		mockUI            *mocks.MockUI
		mockCmdBuilder    *mocks.MockCmdBuilder
		mockCmd           *mocks.MockCmd
		mockExit          *mocks.MockExit
		fakeCliConnection *pluginfakes.FakeCliConnection
		pcfdev            *plugin.Plugin
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockUI(mockCtrl)
		mockCmdBuilder = mocks.NewMockCmdBuilder(mockCtrl)
		mockCmd = mocks.NewMockCmd(mockCtrl)
		mockExit = mocks.NewMockExit(mockCtrl)
		fakeCliConnection = &pluginfakes.FakeCliConnection{}
		pcfdev = &plugin.Plugin{
			UI:         mockUI,
			CmdBuilder: mockCmdBuilder,
			Exit:       mockExit,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Run", func() {
		var home string

		BeforeEach(func() {
			var err error
			home, err = user.GetHome()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when it is called with a good subcommand", func() {
			It("should run the subcommand", func() {
				gomock.InOrder(
					mockCmdBuilder.EXPECT().Cmd("some-command").Return(mockCmd, nil),
					mockCmd.EXPECT().Parse([]string{"some-arg"}),
					mockCmd.EXPECT().Run(),
				)

				pcfdev.Run(fakeCliConnection, []string{"dev", "some-command", "some-arg"})
			})
		})

		Context("when parsing arguments fails", func() {
			It("should print the usage message", func() {
				gomock.InOrder(
					mockCmdBuilder.EXPECT().Cmd("some-command").Return(mockCmd, nil),
					mockCmd.EXPECT().Parse([]string{"some-bad-arg"}).Return(errors.New("some-error")),
				)

				pcfdev.Run(fakeCliConnection, []string{"dev", "some-command", "some-bad-arg"})

				Expect(fakeCliConnection.CliCommandArgsForCall(0)[0]).To(Equal("help"))
				Expect(fakeCliConnection.CliCommandArgsForCall(0)[1]).To(Equal("dev"))
			})
		})

		Context("when running the command fails", func() {
			It("should print the error", func() {
				gomock.InOrder(
					mockCmdBuilder.EXPECT().Cmd("some-command").Return(mockCmd, nil),
					mockCmd.EXPECT().Parse([]string{}),
					mockCmd.EXPECT().Run().Return(errors.New("some-error")),
					mockUI.EXPECT().Failed("Error: some-error."),
					mockExit.EXPECT().Exit(),
				)

				pcfdev.Run(fakeCliConnection, []string{"dev", "some-command"})
			})
		})

		Context("when it is called with no subcommand", func() {
			It("should print the usage message", func() {
				mockCmdBuilder.EXPECT().Cmd("").Return(nil, errors.New(""))

				pcfdev.Run(fakeCliConnection, []string{"dev"})

				Expect(fakeCliConnection.CliCommandArgsForCall(0)[0]).To(Equal("help"))
				Expect(fakeCliConnection.CliCommandArgsForCall(0)[1]).To(Equal("dev"))
			})
		})

		Context("when it is called with an invalid subcommand", func() {
			It("should print the usage message", func() {
				mockCmdBuilder.EXPECT().Cmd("some-bad-subcommand").Return(nil, errors.New(""))
				pcfdev.Run(fakeCliConnection, []string{"dev", "some-bad-subcommand"})

				Expect(fakeCliConnection.CliCommandArgsForCall(0)[0]).To(Equal("help"))
				Expect(fakeCliConnection.CliCommandArgsForCall(0)[1]).To(Equal("dev"))
			})
		})

		Context("when printing the help text fails", func() {
			It("should print an error", func() {
				gomock.InOrder(
					mockCmdBuilder.EXPECT().Cmd("help").Return(nil, errors.New("")),
					mockUI.EXPECT().Failed("Error: some-error."),
					mockExit.EXPECT().Exit(),
				)

				fakeCliConnection.CliCommandReturns(nil, errors.New("some-error"))
				pcfdev.Run(fakeCliConnection, []string{"dev", "help"})
			})
		})
	})

	Describe("Metadata", func() {
		It("should populate version", func() {
			pcfdev.Config = &config.Config{
				Version: &config.Version{
					BuildVersion: "1.2.3",
				}}
			Expect(pcfdev.GetMetadata().Version).To(Equal(cfplugin.VersionType{
				Major: 1,
				Minor: 2,
				Build: 3,
			}))
		})

		Context("when parsing the version fails", func() {
			It("should default to 0.0.0", func() {
				pcfdev.Config = &config.Config{
					Version: &config.Version{
						BuildVersion: "some-bad-version",
					}}
				Expect(pcfdev.GetMetadata().Version).To(Equal(cfplugin.VersionType{
					Major: 0,
					Minor: 0,
					Build: 0,
				}))

				pcfdev.Config = &config.Config{
					Version: &config.Version{
						BuildVersion: "some.bad.version",
					}}
				Expect(pcfdev.GetMetadata().Version).To(Equal(cfplugin.VersionType{
					Major: 0,
					Minor: 0,
					Build: 0,
				}))
			})
		})
	})

	Context("uninstalling plugin", func() {
		It("returns immediately", func() {
			pcfdev.Run(&pluginfakes.FakeCliConnection{}, []string{"CLI-MESSAGE-UNINSTALL"})
		})
	})
})
