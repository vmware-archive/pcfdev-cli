package vm_test

import (
	"errors"

	"github.com/golang/mock/gomock"
	conf "github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vm"
	"github.com/pivotal-cf/pcfdev-cli/vm/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Stopped", func() {
	var (
		mockCtrl *gomock.Controller
		mockUI   *mocks.MockUI
		mockVBox *mocks.MockVBox

		runningVM vm.Running
		config    *conf.VMConfig
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockUI(mockCtrl)
		mockVBox = mocks.NewMockVBox(mockCtrl)
		config = &conf.VMConfig{}

		runningVM = vm.Running{
			VMConfig: &conf.VMConfig{
				Name:    "some-vm",
				Domain:  "some-domain",
				IP:      "some-ip",
				SSHPort: "some-port",
			},

			VBox: mockVBox,
			UI:   mockUI,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Stop", func() {
		It("should stop the vm", func() {
			gomock.InOrder(
				mockUI.EXPECT().Say("Stopping VM..."),
				mockVBox.EXPECT().StopVM(runningVM.VMConfig),
				mockUI.EXPECT().Say("PCF Dev is now stopped"),
			)

			runningVM.Stop()
		})

		Context("when stopped the vm fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Stopping VM..."),
					mockVBox.EXPECT().StopVM(runningVM.VMConfig).Return(errors.New("some-error")),
				)

				Expect(runningVM.Stop()).To(MatchError("failed to stop VM: some-error"))
			})
		})
	})

	Describe("VerifyStartOpts", func() {
		Context("when desired memory is passed", func() {
			It("should return an error", func() {
				Expect(runningVM.VerifyStartOpts(&vm.StartOpts{
					Memory: 4000,
				})).To(MatchError("memory cannot be changed once the vm has been created"))
			})
		})

		Context("when cores is passed", func() {
			It("should return an error", func() {
				Expect(runningVM.VerifyStartOpts(&vm.StartOpts{
					CPUs: 2,
				})).To(MatchError("cores cannot be changed once the vm has been created"))
			})
		})

		Context("when no opts are passed", func() {
			It("should succeed", func() {
				Expect(runningVM.VerifyStartOpts(&vm.StartOpts{})).To(Succeed())
			})
		})

		Context("when services are passed", func() {
			It("should return an error", func() {
				Expect(runningVM.VerifyStartOpts(&vm.StartOpts{
					Services: "redis",
				})).To(MatchError("services cannot be changed once the vm has been created"))
			})
		})
	})

	Describe("Start", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("PCF Dev is running")

			runningVM.Start(&vm.StartOpts{})
		})
	})

	Describe("Status", func() {
		It("should return 'Running' with login instructions", func() {
			Expect(runningVM.Status()).To(Equal("Running\nLogin: cf login -a https://api.some-domain --skip-ssl-validation\nAdmin user => Email: admin / Password: admin\nRegular user => Email: user / Password: pass"))
		})
	})

	Describe("Suspend", func() {
		It("should suspend the vm", func() {
			mockUI.EXPECT().Say("Suspending VM...")
			mockVBox.EXPECT().SuspendVM(runningVM.VMConfig).Return(nil)
			mockUI.EXPECT().Say("PCF Dev is now suspended")

			Expect(runningVM.Suspend()).To(Succeed())
		})

		Context("when suspending the vm fails", func() {
			It("should return an error", func() {
				mockUI.EXPECT().Say("Suspending VM...")
				mockVBox.EXPECT().SuspendVM(runningVM.VMConfig).Return(errors.New("some-error"))

				Expect(runningVM.Suspend()).To(MatchError("failed to suspend VM: some-error"))
			})
		})
	})

	Describe("Resume", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("PCF Dev is running")

			Expect(runningVM.Resume()).To(Succeed())
		})
	})
})
