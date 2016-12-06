package helpers

import "time"

func RemoveDuplicates(collection []string) []string {
	mapping := make(map[string]bool, 0)
	for _, element := range collection {
		mapping[element] = true
	}

	uniqueCollection := make([]string, len(mapping))
	counter := 0
	for k, _ := range mapping {
		uniqueCollection[counter] = k
		counter++
	}
	return uniqueCollection
}

func ExecuteWithAttempts(command func() error, attempts int, delay time.Duration) error {
	var err error
	for attempts > 0 {
		if err = command(); err == nil {
			return nil
		}

		attempts = attempts - 1
		time.Sleep(delay)
	}
	return err
}

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

func IgnoreErrorFrom(_ ...interface{}) {
	// Used as documentation of methods that return errors we are ignoring
	// This makes Errcheck stop complaining.
	// Question usage of this method if you see it!
}
