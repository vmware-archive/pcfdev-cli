go install github.com\pivotal-cf\pcfdev-cli\vendor\github.com\onsi\ginkgo\ginkgo
ginkgo %* -noColor -r %~dp0\..\integration
