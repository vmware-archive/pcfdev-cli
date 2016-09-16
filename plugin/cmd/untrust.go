package cmd

import "github.com/cloudfoundry/cli/cf/flags"

const UNTRUST_ARGS = 0

type UntrustCmd struct {
	CertStore CertStore
}

func (u *UntrustCmd) Parse(args []string) error {
	return parse(flags.New(), args, UNTRUST_ARGS)
}

func (u *UntrustCmd) Run() error {
	return u.CertStore.Unstore()
}
