package requirements_test

import (
	"errors"

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
		mockConfig *mocks.MockConfig
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockSystem = mocks.NewMockSystem(mockCtrl)
		mockConfig = mocks.NewMockConfig(mockCtrl)

		checker = &requirements.Checker{
			System: mockSystem,
			Config: mockConfig,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Check", func() {
		Context("when the free memory is greater than or equal to the minimum memory requirement", func() {
			It("should not return an error", func() {
				mockSystem.EXPECT().FreeMemory().Return(uint64(1048576), nil)
				mockConfig.EXPECT().GetMinMemory().Return(uint64(1))

				Expect(checker.Check()).To(Succeed())
			})
		})

		Context("when the free memory is less than the minimum memory requirement", func() {
			It("should return an error", func() {
				mockSystem.EXPECT().FreeMemory().Return(uint64(1048575), nil)
				mockConfig.EXPECT().GetMinMemory().Return(uint64(1))

				Expect(checker.Check()).To(MatchError("PCF Dev requires 1MB of free memory, this host has 0MB"))
			})
		})

		Context("when the fethcing free memory returns an error", func() {
			It("should return an error", func() {
				mockSystem.EXPECT().FreeMemory().Return(uint64(0), errors.New("some-error"))

				Expect(checker.Check()).To(MatchError("some-error"))
			})
		})
	})
})
