go install github.com\pivotal-cf\pcfdev-cli\vendor\github.com\onsi\ginkgo\ginkgo
ginkgo.exe %* -noColor -skipPackage="integration" -r %~dp0\..
