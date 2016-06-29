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
