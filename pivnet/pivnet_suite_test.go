package pivnet_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestPivNet(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PCF Dev PivNet Suite")
}
