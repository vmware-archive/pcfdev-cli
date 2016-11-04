package cmd

import (
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/cloudfoundry/cli/cf/errors"
)

type ServicesCmd struct {
	VBox VBox
	VMBuilder VMBuilder
	Config *config.Config
}

func (s *ServicesCmd) Parse(args []string) error {
	if !isEmpty(args) {
		return errors.New("uh oh")
	}

	return nil
}

func (s *ServicesCmd) Run() error {
	return nil
}

func isEmpty(items []interface{}) bool {
	return len(items) == 0
}