$scriptPath = split-path -parent $MyInvocation.MyCommand.Definition

go install github.com/pivotal-cf/pcfdev-cli/vendor/github.com/onsi/ginkgo/ginkgo
ginkgo "$@" -r $scriptPath\..\*
