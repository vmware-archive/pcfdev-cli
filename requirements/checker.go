package requirements

type Checker struct {
	MemoryChecker MemoryChecker
}

//go:generate mockgen -package mocks -destination mocks/memory_checker.go github.com/pivotal-cf/pcfdev-cli/requirements MemoryChecker
type MemoryChecker interface {
	Check() error
}

func (c *Checker) Check() error {
	return c.MemoryChecker.Check()
}
