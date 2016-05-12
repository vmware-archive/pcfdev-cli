package ping_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestPing(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PCF Dev Ping Suite")
}
