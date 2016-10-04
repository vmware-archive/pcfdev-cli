package exit

import "os"

type Exit struct{}

func (*Exit) Exit(status int) {
	os.Exit(status)
}
