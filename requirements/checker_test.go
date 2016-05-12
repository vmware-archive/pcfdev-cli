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
		checker           *requirements.Checker
		mockCtrl          *gomock.Controller
		mockMemoryChecker *mocks.MockMemoryChecker
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockMemoryChecker = mocks.NewMockMemoryChecker(mockCtrl)

		checker = &requirements.Checker{
			MemoryChecker: mockMemoryChecker,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Check", func() {
		Context("when the MemoryChecker does not return an error", func() {
			It("should not return an error", func() {
				mockMemoryChecker.EXPECT().Check().Return(nil)

				Expect(checker.Check()).To(Succeed())
			})
		})

		Context("when the MemoryChecker returns an error", func() {
			It("should return an error", func() {
				mockMemoryChecker.EXPECT().Check().Return(errors.New("some-error"))

				Expect(checker.Check()).To(MatchError("some-error"))
			})
		})
	})
})
