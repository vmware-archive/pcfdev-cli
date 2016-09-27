package runner_test

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	rnr "github.com/pivotal-cf/pcfdev-cli/runner"
)

var _ = Describe("CmdRunner", func() {
	var (
		runner *rnr.CmdRunner
	)

	BeforeEach(func() {
		runner = &rnr.CmdRunner{}
	})

	Describe("#Run", func() {
		It("should execute a command and return its output", func() {
			output, err := runner.Run("cmd", "/c", "echo", "some-output")
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.TrimSpace(string(output))).To(Equal("some-output"))
		})

		Context("when there is an error", func() {
			It("should return the error with the output and the arguments", func() {
				_, err := runner.Run("cmd", "/c", "echo some-error && exit 1")
				Expect(strings.TrimSpace(err.Error())).To(Equal("failed to execute 'cmd /c echo some-error && exit 1': exit status 1: some-error"))
			})
		})
	})
})
