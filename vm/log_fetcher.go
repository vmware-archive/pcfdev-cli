package vm

import (
	"bytes"
	"path/filepath"
	"strings"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/config"
)

type ConcreteLogFetcher struct {
	UI     UI
	FS     FS
	SSH    SSH
	Driver Driver

	VMConfig *config.VMConfig
}

type logFile struct {
	command  []string
	reciever string
	filename string
}

const (
	ReceiverGuest = "Guest"
	ReceiverHost  = "Host"
)

func (l *ConcreteLogFetcher) FetchLogs() error {
	logFiles := []logFile{
		logFile{
			command:  []string{"sudo", "cat", "/var/pcfdev/run.log"},
			filename: "run.log",
			reciever: ReceiverGuest,
		},
		logFile{
			command:  []string{"sudo", "cat", "/var/pcfdev/reset.log"},
			filename: "reset.log",
			reciever: ReceiverGuest,
		},
		logFile{
			command:  []string{"sudo", "cat", "/var/log/kern.log"},
			filename: "kern.log",
			reciever: ReceiverGuest,
		},
		logFile{
			command:  []string{"sudo", "cat", "/var/log/dmesg"},
			filename: "dmesg",
			reciever: ReceiverGuest,
		},
		logFile{
			command:  []string{"ifconfig"},
			filename: "ifconfig",
			reciever: ReceiverGuest,
		},
		logFile{
			command:  []string{"route", "-n"},
			filename: "routes",
			reciever: ReceiverGuest,
		},
		logFile{
			command:  []string{"list", "vms"},
			filename: "vm-list",
			reciever: ReceiverHost,
		},
		logFile{
			command:  []string{"showvminfo", l.VMConfig.Name},
			filename: "vm-info",
			reciever: ReceiverHost,
		},
		logFile{
			command:  []string{"list", "hostonlyifs", "--long"},
			filename: "vm-hostonlyifs",
			reciever: ReceiverHost,
		},
	}

	dir, err := l.FS.TempDir()
	if err != nil {
		return err
	}

	for _, logFile := range logFiles {
		switch logFile.reciever {
		case ReceiverGuest:
			output, err := l.SSH.GetSSHOutput(strings.Join(logFile.command, " "), "127.0.0.1", l.VMConfig.SSHPort, 20*time.Second)
			if err != nil {
				return err
			}

			if err := l.FS.Write(filepath.Join(dir, logFile.filename), strings.NewReader(output)); err != nil {
				return err
			}
		case ReceiverHost:
			output, err := l.Driver.VBoxManage(logFile.command...)
			if err != nil {
				return err
			}

			if err := l.FS.Write(filepath.Join(dir, logFile.filename), bytes.NewReader(output)); err != nil {
				return err
			}
		}
	}

	if err := l.FS.Compress("pcfdev-debug", ".", l.getLogFileNames(logFiles, dir)); err != nil {
		return err
	}

	l.UI.Say("Debug logs written to pcfdev-debug.tgz. While some scrubbing has taken place, please remove any remaining sensitive information from these logs before sharing.")
	return nil
}

func (l *ConcreteLogFetcher) getLogFileNames(logFiles []logFile, parentDir string) []string {
	logFileNames := []string{}
	for _, logFile := range logFiles {
		logFileNames = append(logFileNames, filepath.Join(parentDir, logFile.filename))
	}
	return logFileNames
}
