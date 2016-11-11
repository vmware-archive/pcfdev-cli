package cmd_test

import (
	"errors"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/plugin/cmd"
	"github.com/pivotal-cf/pcfdev-cli/plugin/cmd/mocks"
	"github.com/pivotal-cf/pcfdev-cli/vboxdriver"
	"github.com/pivotal-cf/pcfdev-cli/vm"
	vmMocks "github.com/pivotal-cf/pcfdev-cli/vm/mocks"
	"os"
)

var _ = Describe("StartCmd", func() {
	var (
		startCmd         *cmd.StartCmd
		mockCtrl         *gomock.Controller
		mockVMBuilder    *mocks.MockVMBuilder
		mockVBox         *mocks.MockVBox
		mockUI           *mocks.MockUI
		mockVM           *vmMocks.MockVM
		mockStartedVM    *vmMocks.MockVM
		mockAutoTrustCmd *mocks.MockAutoCmd
		mockDownloadCmd  *mocks.MockCmd
		mockTargetCmd    *mocks.MockCmd
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockVMBuilder = mocks.NewMockVMBuilder(mockCtrl)
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockVM = vmMocks.NewMockVM(mockCtrl)
		mockStartedVM = vmMocks.NewMockVM(mockCtrl)
		mockDownloadCmd = mocks.NewMockCmd(mockCtrl)
		mockAutoTrustCmd = mocks.NewMockAutoCmd(mockCtrl)
		mockTargetCmd = mocks.NewMockCmd(mockCtrl)
		mockUI = mocks.NewMockUI(mockCtrl)
		startCmd = &cmd.StartCmd{
			VBox:      mockVBox,
			VMBuilder: mockVMBuilder,
			Config: &config.Config{
				DefaultVMName: "some-default-vm-name",
			},
			Opts:         &vm.StartOpts{},
			DownloadCmd:  mockDownloadCmd,
			AutoTrustCmd: mockAutoTrustCmd,
			TargetCmd:    mockTargetCmd,
			UI:           mockUI,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Parse", func() {
		Context("when flags are passed", func() {
			It("should set start options", func() {
				Expect(startCmd.Parse([]string{
					"-c", "2",
					"-k",
					"-m", "3456",
					"-n",
					"-o", "some-ova-path",
					"-r", "some-private-registry,some-other-private-registry",
					"-s", "some-service,some-other-service",
					"-t",
					"-i", "some-ip",
					"-d", "some-domain",
				})).To(Succeed())

				Expect(startCmd.Opts.CPUs).To(Equal(2))
				Expect(startCmd.Opts.Memory).To(Equal(uint64(3456)))
				Expect(startCmd.Opts.NoProvision).To(BeTrue())
				Expect(startCmd.Opts.OVAPath).To(Equal("some-ova-path"))
				Expect(startCmd.Opts.Registries).To(Equal("some-private-registry,some-other-private-registry"))
				Expect(startCmd.Opts.Services).To(Equal("some-service,some-other-service"))
				Expect(startCmd.Opts.Target).To(BeTrue())
				Expect(startCmd.Opts.Domain).To(Equal("some-domain"))
				Expect(startCmd.Opts.IP).To(Equal("some-ip"))
				Expect(startCmd.Opts.MasterPassword).To(Equal(""))
			})
		})

		Context("when no flags are passed", func() {
			It("should set start options", func() {
				Expect(startCmd.Parse([]string{})).To(Succeed())
				Expect(startCmd.Opts.CPUs).To(Equal(0))
				Expect(startCmd.Opts.Memory).To(Equal(uint64(0)))
				Expect(startCmd.Opts.NoProvision).To(BeFalse())
				Expect(startCmd.Opts.OVAPath).To(BeEmpty())
				Expect(startCmd.Opts.Registries).To(BeEmpty())
				Expect(startCmd.Opts.Services).To(BeEmpty())
				Expect(startCmd.Opts.Target).To(BeFalse())
				Expect(startCmd.Opts.Domain).To(BeEmpty())
				Expect(startCmd.Opts.IP).To(BeEmpty())
				Expect(startCmd.Opts.MasterPassword).To(BeEmpty())
			})
		})

		Context("when the PCFDEV_PASSWORD env var is set", func() {
			var savedPassword string

			BeforeEach(func() {
				savedPassword = os.Getenv("PCFDEV_PASSWORD")
				os.Setenv("PCFDEV_PASSWORD", "some-master-password")
			})

			AfterEach(func() {
				os.Setenv("PCFDEV_PASSWORD", savedPassword)
			})

			It("the -x flag uses the env var instead of prompting", func() {
				Expect(startCmd.Parse([]string{
					"-x",
				})).To(Succeed())
				Expect(startCmd.Opts.MasterPassword).To(Equal("some-master-password"))
			})

			It("the env var is ignored without the -x flag", func() {
				Expect(startCmd.Parse([]string{})).To(Succeed())
				Expect(startCmd.Opts.MasterPassword).To(BeEmpty())
			})
		})

		Context("when the -x flag is specified", func() {
			It("should prompt the user for a password twice", func() {
				mockUI.EXPECT().AskForPassword("Choose master password").Return("some-master-password")
				mockUI.EXPECT().AskForPassword("Confirm master password").Return("some-master-password")

				Expect(startCmd.Parse([]string{
					"-x",
				})).To(Succeed())

				Expect(startCmd.Opts.MasterPassword).To(Equal("some-master-password"))
			})

			It("does not allow an empty password", func() {
				mockUI.EXPECT().AskForPassword("Choose master password").Return("")
				mockUI.EXPECT().AskForPassword("Confirm master password").Return("")

				Expect(startCmd.Parse([]string{
					"-x",
				})).To(MatchError("password cannot be empty"))
			})

			It("does not allow an mismatched password", func() {
				mockUI.EXPECT().AskForPassword("Choose master password").Return("some-master-password")
				mockUI.EXPECT().AskForPassword("Confirm master password").Return("some-bad-password")

				Expect(startCmd.Parse([]string{
					"-x",
				})).To(MatchError("passwords do not match"))
			})
		})

		Context("when an unknown flag is passed", func() {
			It("should return an error", func() {
				Expect(startCmd.Parse(
					[]string{"-b", "some-bad-flag"})).NotTo(Succeed())
			})
		})

		Context("when an unknown argument is passed", func() {
			It("should return an error", func() {
				Expect(startCmd.Parse(
					[]string{"some-bad-argument"})).NotTo(Succeed())
			})
		})
	})

	Describe("Run", func() {
		BeforeEach(func() {
			startCmd.Parse([]string{})
		})

		Context("when starting the default ova", func() {
			It("should validate start options and start the VM", func() {
				startOpts := &vm.StartOpts{
					Memory: uint64(3456),
					CPUs:   2,
				}
				startCmd.Opts = startOpts
				gomock.InOrder(
					mockVBox.EXPECT().Version().Return(&vboxdriver.VBoxDriverVersion{Major: 5}, nil),
					mockVBox.EXPECT().GetVMName().Return("", nil),
					mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().VerifyStartOpts(startOpts),
					mockDownloadCmd.EXPECT().Run(),
					mockVM.EXPECT().Start(startOpts),
				)

				Expect(startCmd.Run()).To(Succeed())
			})

			Context("when the trust option is passed", func() {
				It("should trust the VM certificate after starting", func() {
					startCmd.Parse([]string{"-k"})

					gomock.InOrder(
						mockVBox.EXPECT().Version().Return(&vboxdriver.VBoxDriverVersion{Major: 5}, nil),
						mockVBox.EXPECT().GetVMName().Return("", nil),
						mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
						mockVM.EXPECT().VerifyStartOpts(&vm.StartOpts{}),
						mockDownloadCmd.EXPECT().Run(),
						mockVM.EXPECT().Start(&vm.StartOpts{}),
						mockAutoTrustCmd.EXPECT().Run(),
					)

					Expect(startCmd.Run()).To(Succeed())
				})
			})

			Context("when the target option is passed", func() {
				It("should target PCF Dev after starting", func() {
					startCmd.Parse([]string{"-t"})

					gomock.InOrder(
						mockVBox.EXPECT().Version().Return(&vboxdriver.VBoxDriverVersion{Major: 5}, nil),
						mockVBox.EXPECT().GetVMName().Return("", nil),
						mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
						mockVM.EXPECT().VerifyStartOpts(&vm.StartOpts{Target: true}),
						mockDownloadCmd.EXPECT().Run(),
						mockVM.EXPECT().Start(&vm.StartOpts{Target: true}),
						mockTargetCmd.EXPECT().Run(),
					)

					Expect(startCmd.Run()).To(Succeed())
				})
			})

			Context("when targeting PCF Dev fails", func() {
				It("should return the error", func() {
					startCmd.Parse([]string{"-t"})

					gomock.InOrder(
						mockVBox.EXPECT().Version().Return(&vboxdriver.VBoxDriverVersion{Major: 5}, nil),
						mockVBox.EXPECT().GetVMName().Return("", nil),
						mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
						mockVM.EXPECT().VerifyStartOpts(&vm.StartOpts{Target: true}),
						mockDownloadCmd.EXPECT().Run(),
						mockVM.EXPECT().Start(&vm.StartOpts{Target: true}),
						mockTargetCmd.EXPECT().Run().Return(errors.New("some-error")),
					)

					Expect(startCmd.Run()).To(MatchError("some-error"))
				})
			})

			Context("when targeting PCF Dev and trusting VM certificates", func() {
				It("should target PCF Dev and trust the VM certificates", func() {
					startCmd.Parse([]string{"-t", "-k"})

					gomock.InOrder(
						mockVBox.EXPECT().Version().Return(&vboxdriver.VBoxDriverVersion{Major: 5}, nil),
						mockVBox.EXPECT().GetVMName().Return("", nil),
						mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
						mockVM.EXPECT().VerifyStartOpts(&vm.StartOpts{Target: true}),
						mockDownloadCmd.EXPECT().Run(),
						mockVM.EXPECT().Start(&vm.StartOpts{Target: true}),
						mockAutoTrustCmd.EXPECT().Run(),
						mockTargetCmd.EXPECT().Run(),
					)

					Expect(startCmd.Run()).To(Succeed())
				})
			})

			Context("when virtualbox version is too old", func() {
				It("should tell the user to upgrade virtualbox", func() {
					mockVBox.EXPECT().Version().Return(&vboxdriver.VBoxDriverVersion{Major: 4}, nil)

					Expect(startCmd.Run()).To(MatchError("please install Virtualbox version 5 or greater"))
				})
			})

			Context("when there is an error retrieving the virtualbox version", func() {
				It("should return the error", func() {
					mockVBox.EXPECT().Version().Return(nil, errors.New("some-error"))

					Expect(startCmd.Run()).To(MatchError("some-error"))
				})
			})

			Context("when there is an old vm present", func() {
				It("should tell the user to destroy pcfdev", func() {
					mockVBox.EXPECT().Version().Return(&vboxdriver.VBoxDriverVersion{Major: 5}, nil)
					mockVBox.EXPECT().GetVMName().Return("some-old-vm-name", nil)

					Expect(startCmd.Run()).To(MatchError("old version of PCF Dev already running, please run `cf dev destroy` to continue"))
				})
			})

			Context("when there is an error getting the VM name", func() {
				It("should return the error", func() {
					mockVBox.EXPECT().Version().Return(&vboxdriver.VBoxDriverVersion{Major: 5}, nil)
					mockVBox.EXPECT().GetVMName().Return("", errors.New("some-error"))

					Expect(startCmd.Run()).To(MatchError("some-error"))
				})
			})

			Context("when it fails to get VM", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockVBox.EXPECT().Version().Return(&vboxdriver.VBoxDriverVersion{Major: 5}, nil),
						mockVBox.EXPECT().GetVMName().Return("", nil),
						mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(nil, errors.New("some-error")),
					)
					Expect(startCmd.Run()).To(MatchError("some-error"))
				})
			})

			Context("when verifying start options fails", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockVBox.EXPECT().Version().Return(&vboxdriver.VBoxDriverVersion{Major: 5}, nil),
						mockVBox.EXPECT().GetVMName().Return("", nil),
						mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
						mockVM.EXPECT().VerifyStartOpts(&vm.StartOpts{}).Return(errors.New("some-error")),
					)

					Expect(startCmd.Run()).To(MatchError("some-error"))
				})
			})

			Context("when the OVA fails to download", func() {
				It("should print an error message", func() {
					gomock.InOrder(
						mockVBox.EXPECT().Version().Return(&vboxdriver.VBoxDriverVersion{Major: 5}, nil),
						mockVBox.EXPECT().GetVMName().Return("", nil),
						mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
						mockVM.EXPECT().VerifyStartOpts(&vm.StartOpts{}),
						mockDownloadCmd.EXPECT().Run().Return(errors.New("some-error")),
					)

					Expect(startCmd.Run()).To(MatchError("some-error"))
				})
			})

			Context("when it fails to start VM", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockVBox.EXPECT().Version().Return(&vboxdriver.VBoxDriverVersion{Major: 5}, nil),
						mockVBox.EXPECT().GetVMName().Return("", nil),
						mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
						mockVM.EXPECT().VerifyStartOpts(&vm.StartOpts{}),
						mockDownloadCmd.EXPECT().Run(),
						mockVM.EXPECT().Start(&vm.StartOpts{}).Return(errors.New("some-error")),
					)

					Expect(startCmd.Run()).To(MatchError("some-error"))
				})
			})

			Context("when there is an error trusting the VM certificates", func() {
				It("should return the error", func() {
					startCmd.Parse([]string{"-k"})

					gomock.InOrder(
						mockVBox.EXPECT().Version().Return(&vboxdriver.VBoxDriverVersion{Major: 5}, nil),
						mockVBox.EXPECT().GetVMName().Return("", nil),
						mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
						mockVM.EXPECT().VerifyStartOpts(&vm.StartOpts{}),
						mockDownloadCmd.EXPECT().Run(),
						mockVM.EXPECT().Start(&vm.StartOpts{}),
						mockAutoTrustCmd.EXPECT().Run().Return(errors.New("some-error")),
					)

					Expect(startCmd.Run()).To(MatchError("some-error"))
				})
			})
		})

		Context("when starting a custom ova", func() {
			It("should start the custom ova", func() {
				startOpts := &vm.StartOpts{
					OVAPath: "some-custom-ova",
				}
				startCmd.Opts = startOpts
				gomock.InOrder(
					mockVBox.EXPECT().Version().Return(&vboxdriver.VBoxDriverVersion{Major: 5}, nil),
					mockVBox.EXPECT().GetVMName().Return("", nil),
					mockVMBuilder.EXPECT().VM("pcfdev-custom").Return(mockVM, nil),
					mockVM.EXPECT().VerifyStartOpts(startOpts),
					mockVM.EXPECT().Start(startOpts),
				)

				Expect(startCmd.Run()).To(Succeed())
			})

			Context("when the custom VM is already present and OVAPath is not set", func() {
				It("should start the custom VM", func() {
					gomock.InOrder(
						mockVBox.EXPECT().Version().Return(&vboxdriver.VBoxDriverVersion{Major: 5}, nil),
						mockVBox.EXPECT().GetVMName().Return("pcfdev-custom", nil),
						mockVMBuilder.EXPECT().VM("pcfdev-custom").Return(mockVM, nil),
						mockVM.EXPECT().VerifyStartOpts(&vm.StartOpts{}),
						mockVM.EXPECT().Start(&vm.StartOpts{}),
					)
					Expect(startCmd.Run()).To(Succeed())
				})
			})

			Context("when the custom VM is already present and OVAPath is set", func() {
				It("should start the custom OVA", func() {
					startOpts := &vm.StartOpts{
						OVAPath: "some-custom-ova",
					}
					startCmd.Opts = startOpts
					gomock.InOrder(
						mockVBox.EXPECT().Version().Return(&vboxdriver.VBoxDriverVersion{Major: 5}, nil),
						mockVBox.EXPECT().GetVMName().Return("pcfdev-custom", nil),
						mockVMBuilder.EXPECT().VM("pcfdev-custom").Return(mockVM, nil),
						mockVM.EXPECT().VerifyStartOpts(startOpts),
						mockVM.EXPECT().Start(startOpts),
					)
					Expect(startCmd.Run()).To(Succeed())
				})
			})

			Context("when the default VM is present", func() {
				It("should return an error", func() {
					startOpts := &vm.StartOpts{
						OVAPath: "some-custom-ova",
					}
					startCmd.Opts = startOpts
					mockVBox.EXPECT().Version().Return(&vboxdriver.VBoxDriverVersion{Major: 5}, nil)
					mockVBox.EXPECT().GetVMName().Return("some-default-vm-name", nil)
					Expect(startCmd.Run()).To(MatchError("you must destroy your existing VM to use a custom OVA"))
				})
			})

			Context("when an old VM is present", func() {
				It("should return an error", func() {
					startOpts := &vm.StartOpts{
						OVAPath: "some-custom-ova",
					}
					startCmd.Opts = startOpts
					mockVBox.EXPECT().Version().Return(&vboxdriver.VBoxDriverVersion{Major: 5}, nil)
					mockVBox.EXPECT().GetVMName().Return("some-old-vm-name", nil)
					Expect(startCmd.Run()).To(MatchError("you must destroy your existing VM to use a custom OVA"))
				})
			})
		})

		Context("when the provision option is specified", func() {
			It("should provision the VM", func() {
				startCmd.Parse([]string{"-p"})

				gomock.InOrder(
					mockVBox.EXPECT().Version().Return(&vboxdriver.VBoxDriverVersion{Major: 5}, nil),
					mockVBox.EXPECT().GetVMName().Return("", nil),
					mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().Provision(&vm.StartOpts{}),
				)

				Expect(startCmd.Run()).To(Succeed())
			})
		})

		Context("when provisioning fails", func() {
			It("return an error", func() {
				startCmd.Parse([]string{"-p"})

				gomock.InOrder(
					mockVBox.EXPECT().Version().Return(&vboxdriver.VBoxDriverVersion{Major: 5}, nil),
					mockVBox.EXPECT().GetVMName().Return("", nil),
					mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().Provision(&vm.StartOpts{}).Return(errors.New("some-error")),
				)

				Expect(startCmd.Run()).To(MatchError("some-error"))
			})
		})
	})
})
