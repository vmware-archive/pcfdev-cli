package runner_test

import (
	"runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	rnr "github.com/pivotal-cf/pcfdev-cli/runner"
)

var _ = Describe("CmdRunner", func() {
	var (
		runner *rnr.CmdRunner
	)

	BeforeEach(func() {
		if runtime.GOOS == "windows" {
			Skip("This test is not appropriate for the windows OS")
		}

		runner = &rnr.CmdRunner{}
	})

	Describe("#Run", func() {
		It("should execute a command and return its output", func() {
			Expect(runner.Run("echo", "-n", "some-output")).To(Equal([]byte("some-output")))
			Expect(runner.Run("bash", "-c", ">&2 echo -n some-output")).To(Equal([]byte("some-output")))
		})

		Context("when there is an error", func() {
			It("should return the error", func() {
				_, err := runner.Run("/some/bad/binary")
				Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
			})
		})
	})
})
