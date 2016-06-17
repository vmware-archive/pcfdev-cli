package helpers

import "time"

func ExecuteWithTimeout(command func() error, timeout time.Duration, delay time.Duration) error {
	timeoutChan := time.After(timeout)
	var err error

	for {
		select {
		case <-timeoutChan:
			return err
		default:
			if err = command(); err == nil {
				return nil
			}
			time.Sleep(delay)
		}
	}
}
