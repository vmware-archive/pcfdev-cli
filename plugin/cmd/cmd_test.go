package cmd_test

import (
	"os"

	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/cloudfoundry/cli/cf/trace"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/downloader"
	"github.com/pivotal-cf/pcfdev-cli/fs"
	"github.com/pivotal-cf/pcfdev-cli/pivnet"
	"github.com/pivotal-cf/pcfdev-cli/plugin/cmd"
	"github.com/pivotal-cf/pcfdev-cli/ui"
	"github.com/pivotal-cf/pcfdev-cli/vbox"
	"github.com/pivotal-cf/pcfdev-cli/vm"
)

var _ = Describe("Builder", func() {
	Describe("Cmd", func() {
		var builder *cmd.Builder
		BeforeEach(func() {
			builder = &cmd.Builder{
				VBox:              &vbox.VBox{},
				DownloaderFactory: &downloader.DownloaderFactory{},
				FS:                &fs.FS{},
				UI: terminal.NewUI(
					os.Stdin,
					os.Stdout,
					terminal.NewTeePrinter(os.Stdout),
					trace.NewWriterPrinter(os.Stdout, true),
				),
				VMBuilder: &vm.VBoxBuilder{},
				Config:    &config.Config{},
				EULAUI:    &ui.UI{},
				Client:    &pivnet.Client{},
			}
		})

		Context("when it is passed destroy", func() {
			It("should return a destroy command", func() {
				destroyCmd, err := builder.Cmd("destroy")
				Expect(err).NotTo(HaveOccurred())

				switch c := destroyCmd.(type) {
				case *cmd.DestroyCmd:
					Expect(c.VBox).To(BeIdenticalTo(builder.VBox))
					Expect(c.UI).To(BeIdenticalTo(builder.UI))
					Expect(c.FS).To(BeIdenticalTo(builder.FS))
					Expect(c.Config).To(BeIdenticalTo(builder.Config))
				default:
					Fail("wrong type")
				}
			})
		})

		Context("when it is passed download", func() {
			It("should return a download command", func() {
				downloadCmd, err := builder.Cmd("download")
				Expect(err).NotTo(HaveOccurred())

				switch c := downloadCmd.(type) {
				case *cmd.DownloadCmd:
					Expect(c.VBox).To(BeIdenticalTo(builder.VBox))
					Expect(c.UI).To(BeIdenticalTo(builder.UI))
					Expect(c.EULAUI).To(BeIdenticalTo(builder.EULAUI))
					Expect(c.Client).To(BeIdenticalTo(builder.Client))
					Expect(c.DownloaderFactory).To(BeIdenticalTo(builder.DownloaderFactory))
					Expect(c.FS).To(BeIdenticalTo(builder.FS))
					Expect(c.Config).To(BeIdenticalTo(builder.Config))
				default:
					Fail("wrong type")
				}
			})
		})

		Context("when it is passed import", func() {
			It("should return an import command", func() {
				importCmd, err := builder.Cmd("import")
				Expect(err).NotTo(HaveOccurred())

				switch c := importCmd.(type) {
				case *cmd.ImportCmd:
					Expect(c.DownloaderFactory).To(BeIdenticalTo(builder.DownloaderFactory))
					Expect(c.UI).To(BeIdenticalTo(builder.UI))
					Expect(c.FS).To(BeIdenticalTo(builder.FS))
					Expect(c.Config).To(BeIdenticalTo(builder.Config))
				default:
					Fail("wrong type")
				}
			})
		})

		Context("when it is passed resume", func() {
			It("should return a resume command", func() {
				resumeCmd, err := builder.Cmd("resume")
				Expect(err).NotTo(HaveOccurred())

				switch c := resumeCmd.(type) {
				case *cmd.ResumeCmd:
					Expect(c.VBox).To(BeIdenticalTo(builder.VBox))
					Expect(c.VMBuilder).To(BeIdenticalTo(builder.VMBuilder))
					Expect(c.Config).To(BeIdenticalTo(builder.Config))
				default:
					Fail("wrong type")
				}
			})
		})

		Context("when it is passed start", func() {
			It("should return a start command", func() {
				startCmd, err := builder.Cmd("start")
				Expect(err).NotTo(HaveOccurred())

				switch c := startCmd.(type) {
				case *cmd.StartCmd:
					Expect(c.VBox).To(BeIdenticalTo(builder.VBox))
					Expect(c.VMBuilder).To(BeIdenticalTo(builder.VMBuilder))
					Expect(c.Config).To(BeIdenticalTo(builder.Config))
					Expect(c.DownloadCmd).To(Equal(&cmd.DownloadCmd{
						VBox:              builder.VBox,
						UI:                builder.UI,
						EULAUI:            builder.EULAUI,
						Client:            builder.Client,
						DownloaderFactory: builder.DownloaderFactory,
						FS:                builder.FS,
						Config:            builder.Config,
					}))
				default:
					Fail("wrong type")
				}
			})
		})

		Context("when it is passed status", func() {
			It("should return a status command", func() {
				statusCmd, err := builder.Cmd("status")
				Expect(err).NotTo(HaveOccurred())

				switch c := statusCmd.(type) {
				case *cmd.StatusCmd:
					Expect(c.VBox).To(BeIdenticalTo(builder.VBox))
					Expect(c.VMBuilder).To(BeIdenticalTo(builder.VMBuilder))
					Expect(c.Config).To(BeIdenticalTo(builder.Config))
					Expect(c.UI).To(BeIdenticalTo(builder.UI))
				default:
					Fail("wrong type")
				}
			})
		})

		Context("when it is passed stop", func() {
			It("should return a stop command", func() {
				stopCmd, err := builder.Cmd("stop")
				Expect(err).NotTo(HaveOccurred())

				switch c := stopCmd.(type) {
				case *cmd.StopCmd:
					Expect(c.VBox).To(BeIdenticalTo(builder.VBox))
					Expect(c.VMBuilder).To(BeIdenticalTo(builder.VMBuilder))
					Expect(c.Config).To(BeIdenticalTo(builder.Config))
				default:
					Fail("wrong type")
				}
			})
		})

		Context("when it is passed suspend", func() {
			It("should return a suspend command", func() {
				suspendCmd, err := builder.Cmd("suspend")
				Expect(err).NotTo(HaveOccurred())

				switch c := suspendCmd.(type) {
				case *cmd.SuspendCmd:
					Expect(c.VBox).To(BeIdenticalTo(builder.VBox))
					Expect(c.VMBuilder).To(BeIdenticalTo(builder.VMBuilder))
					Expect(c.Config).To(BeIdenticalTo(builder.Config))
				default:
					Fail("wrong type")
				}
			})
		})

		Context("when it is passed version", func() {
			It("should return a version command", func() {
				versionCmd, err := builder.Cmd("version")
				Expect(err).NotTo(HaveOccurred())

				switch c := versionCmd.(type) {
				case *cmd.VersionCmd:
					Expect(c.Config).To(BeIdenticalTo(builder.Config))
				default:
					Fail("wrong type")
				}
			})
		})

		Context("when it is passed --version", func() {
			It("should return a version command", func() {
				versionCmd, err := builder.Cmd("--version")
				Expect(err).NotTo(HaveOccurred())

				switch c := versionCmd.(type) {
				case *cmd.VersionCmd:
					Expect(c.Config).To(BeIdenticalTo(builder.Config))
					Expect(c.UI).To(BeIdenticalTo(builder.UI))
				default:
					Fail("wrong type")
				}
			})
		})

		Context("when it is passed debug", func() {
			It("should return a debug command", func() {
				debugCmd, err := builder.Cmd("debug")
				Expect(err).NotTo(HaveOccurred())

				switch c := debugCmd.(type) {
				case *cmd.DebugCmd:
					Expect(c.VBox).To(BeIdenticalTo(builder.VBox))
					Expect(c.VMBuilder).To(BeIdenticalTo(builder.VMBuilder))
					Expect(c.Config).To(BeIdenticalTo(builder.Config))
				default:
					Fail("wrong type")
				}
			})
		})

		Context("when is is passed 'trust'", func() {
			It("should return a trust command", func() {
				trustCmd, err := builder.Cmd("trust")
				Expect(err).NotTo(HaveOccurred())

				switch c := trustCmd.(type) {
				case *cmd.TrustCmd:
					Expect(c.VBox).To(BeIdenticalTo(builder.VBox))
					Expect(c.VMBuilder).To(BeIdenticalTo(builder.VMBuilder))
					Expect(c.Config).To(BeIdenticalTo(builder.Config))
				default:
					Fail("wrong type")
				}
			})
		})

		Context("when it is passed an unknown subcommand", func() {
			It("should return an error", func() {
				_, err := builder.Cmd("some-bad-subcommand")
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
