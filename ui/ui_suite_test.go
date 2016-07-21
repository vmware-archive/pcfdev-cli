package ui_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestUI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PCF Dev UI Suite")
}
