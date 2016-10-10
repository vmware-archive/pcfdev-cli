package debug

import (
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/config"
)

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/pivotal-cf/pcfdev-cli/debug FS
type FS interface {
	Read(path string) (contents []byte, err error)
	Write(path string, contents io.Reader, append bool) error
	Compress(name string, path string, contentPaths []string) error
	TempDir() (tempDir string, err error)
}

//go:generate mockgen -package mocks -destination mocks/ssh.go github.com/pivotal-cf/pcfdev-cli/debug SSH
type SSH interface {
	GetSSHOutput(command string, ip string, port string, privateKey string, timeout time.Duration) (combinedOutput string, err error)
}

//go:generate mockgen -package mocks -destination mocks/driver.go github.com/pivotal-cf/pcfdev-cli/debug Driver
type Driver interface {
	VBoxManage(arg ...string) (output []byte, err error)
}

type LogFetcher struct {
	FS     FS
	SSH    SSH
	Driver Driver

	VMConfig *config.VMConfig
	Config   *config.Config
}

type logFile struct {
	command   []string
	reciever  string
	filename  string
	sensitive bool
}

const (
	ReceiverGuest = "Guest"
	ReceiverHost  = "Host"
)

func (l *LogFetcher) FetchLogs() error {
	logFiles := []logFile{
		logFile{
			command:   []string{"sudo", "cat", "/var/pcfdev/provision.log"},
			filename:  "provision.log",
			reciever:  ReceiverGuest,
			sensitive: true,
		},
		logFile{
			command:   []string{"sudo", "cat", "/var/pcfdev/reset.log"},
			filename:  "reset.log",
			reciever:  ReceiverGuest,
			sensitive: true,
		},
		logFile{
			command:   []string{"sudo", "cat", "/var/log/kern.log"},
			filename:  "kern.log",
			reciever:  ReceiverGuest,
			sensitive: true,
		},
		logFile{
			command:   []string{"sudo", "cat", "/var/log/dmesg"},
			filename:  "dmesg",
			reciever:  ReceiverGuest,
			sensitive: true,
		},
		logFile{
			command:   []string{"ifconfig"},
			filename:  "ifconfig",
			reciever:  ReceiverGuest,
			sensitive: false,
		},
		logFile{
			command:   []string{"route", "-n"},
			filename:  "routes",
			reciever:  ReceiverGuest,
			sensitive: false,
		},
		logFile{
			command:   []string{"list", "vms", "--long"},
			filename:  "vm-list",
			reciever:  ReceiverHost,
			sensitive: false,
		},
		logFile{
			command:   []string{"showvminfo", l.VMConfig.Name},
			filename:  "vm-info",
			reciever:  ReceiverHost,
			sensitive: false,
		},
		logFile{
			command:   []string{"list", "hostonlyifs", "--long"},
			filename:  "vm-hostonlyifs",
			reciever:  ReceiverHost,
			sensitive: false,
		},
	}

	sensitiveInformationScrubber := &SensitiveInformationScrubber{}

	privateKeyBytes, err := l.FS.Read(l.Config.PrivateKeyPath)
	if err != nil {
		return err
	}

	dir, err := l.FS.TempDir()
	if err != nil {
		return err
	}

	for _, logFile := range logFiles {
		switch logFile.reciever {
		case ReceiverGuest:
			output, err := l.SSH.GetSSHOutput(strings.Join(logFile.command, " "), "127.0.0.1", l.VMConfig.SSHPort, string(privateKeyBytes), 20*time.Second)
			if err != nil {
				return err
			}

			if logFile.sensitive {
				output = sensitiveInformationScrubber.Scrub(output)
			}

			if err := l.FS.Write(
				filepath.Join(dir, logFile.filename),
				strings.NewReader(output),
				false,
			); err != nil {
				return err
			}
		case ReceiverHost:
			output, err := l.Driver.VBoxManage(logFile.command...)
			if err != nil {
				return err
			}

			scrubbedOutput := string(output)
			if logFile.sensitive {
				scrubbedOutput = sensitiveInformationScrubber.Scrub(scrubbedOutput)
			}

			if err := l.FS.Write(
				filepath.Join(dir, logFile.filename),
				strings.NewReader(scrubbedOutput),
				false,
			); err != nil {
				return err
			}
		}
	}

	if err := l.FS.Compress("pcfdev-debug", ".", l.getLogFileNames(logFiles, dir)); err != nil {
		return err
	}

	return nil
}

func (l *LogFetcher) getLogFileNames(logFiles []logFile, parentDir string) []string {
	logFileNames := []string{}
	for _, logFile := range logFiles {
		logFileNames = append(logFileNames, filepath.Join(parentDir, logFile.filename))
	}
	return logFileNames
}
