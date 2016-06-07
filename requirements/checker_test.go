package requirements_test

import (
	"errors"

	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/requirements"
	"github.com/pivotal-cf/pcfdev-cli/requirements/mocks"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Checker", func() {
	var (
		checker    *requirements.Checker
		mockCtrl   *gomock.Controller
		mockSystem *mocks.MockSystem
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockSystem = mocks.NewMockSystem(mockCtrl)

		checker = &requirements.Checker{
			System: mockSystem,
			Config: &config.Config{
				MinMemory: uint64(3),
			},
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("CheckMemory", func() {
		Context("when desired memory is less than free memory and greater than minumum memory", func() {
			It("should return an error", func() {
				mockSystem.EXPECT().FreeMemory().Return(uint64(10*1048576), nil)

				Expect(checker.CheckMemory(uint64(4))).To(Succeed())
			})
		})

		Context("when desired memory is less than minumum memory requirement", func() {
			It("should return an error", func() {
				Expect(checker.CheckMemory(uint64(1))).To(MatchError("PCF Dev requires at least 3 MB of memory to run."))
			})
		})

		Context("when desired memory is greater than the free memory", func() {
			It("should return an error", func() {
				mockSystem.EXPECT().FreeMemory().Return(uint64(1*1048576), nil)

				Expect(checker.CheckMemory(uint64(4))).To(Equal(&requirements.NotEnoughMemoryError{
					DesiredMemory: uint64(4),
					FreeMemory:    uint64(1),
				}))
			})
		})

		Context("when the fetching free memory returns an error", func() {
			It("should return an error", func() {
				mockSystem.EXPECT().FreeMemory().Return(uint64(0), errors.New("some-error"))

				Expect(checker.CheckMemory(uint64(3))).To(MatchError("some-error"))
			})
		})

	})

	Describe("CheckMinMemory", func() {
		Context("when minimum memory is less than free memory", func() {
			It("should return an error", func() {
				mockSystem.EXPECT().FreeMemory().Return(uint64(2*1048576), nil)

				Expect(checker.CheckMinMemory()).To(Equal(&requirements.NotEnoughMemoryError{
					DesiredMemory: uint64(3),
					FreeMemory:    uint64(2),
				}))
			})
		})
	})
})
