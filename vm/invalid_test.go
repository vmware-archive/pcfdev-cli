package vm_test

import (
	"errors"

	"github.com/pivotal-cf/pcfdev-cli/vm"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Invalid", func() {
	var invalid vm.Invalid

	BeforeEach(func() {
		invalid = vm.Invalid{
			Err: errors.New("some-error"),
		}
	})

	Describe("Stop", func() {
		It("should say a message", func() {
			Expect(invalid.Stop()).To(MatchError("some-error.\nPCF Dev is in an invalid state. Please run 'cf dev destroy'"))
		})
	})

	Describe("VerifyStartOpts", func() {
		It("should succeed", func() {
			Expect(invalid.VerifyStartOpts(
				&vm.StartOpts{},
			)).To(MatchError("some-error.\nPCF Dev is in an invalid state. Please run 'cf dev destroy'"))
		})
	})

	Describe("Start", func() {
		It("should start vm", func() {
			Expect(invalid.Start(&vm.StartOpts{})).To(MatchError("some-error.\nPCF Dev is in an invalid state. Please run 'cf dev destroy'"))
		})
	})

	Describe("Status", func() {
		It("should return 'Status'", func() {
			Expect(invalid.Status()).To(Equal("PCF Dev is in an invalid state. Please run 'cf dev destroy'"))
		})
	})

	Describe("Suspend", func() {
		It("should say a message", func() {
			Expect(invalid.Suspend()).To(MatchError("some-error.\nPCF Dev is in an invalid state. Please run 'cf dev destroy'"))
		})
	})

	Describe("Resume", func() {
		It("should say a message", func() {
			Expect(invalid.Resume()).To(MatchError("some-error.\nPCF Dev is in an invalid state. Please run 'cf dev destroy'"))
		})
	})

	Describe("GetDebugLogs", func() {
		It("should say a message", func() {
			Expect(invalid.GetDebugLogs()).To(MatchError("some-error.\nPCF Dev is in an invalid state. Please run 'cf dev destroy'"))
		})
	})

	Describe("Trust", func() {
		It("should say a message", func() {
			Expect(invalid.Trust(&vm.StartOpts{})).To(MatchError("some-error.\nPCF Dev is in an invalid state. Please run 'cf dev destroy'"))
		})
	})

	Describe("Target", func() {
		It("should say a message", func() {
			Expect(invalid.Target(false)).To(MatchError("some-error.\nPCF Dev is in an invalid state. Please run 'cf dev destroy'"))
		})
	})

	Describe("SSH", func() {
		It("should say a message", func() {
			Expect(invalid.SSH()).To(MatchError("some-error.\nPCF Dev is in an invalid state. Please run 'cf dev destroy'"))
		})
	})
})
