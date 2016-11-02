package exit

import "os"

type Exit struct{}

func (*Exit) Exit() {
	someStatusCodeThatCfCliNeverReads := 1
	os.Exit(someStatusCodeThatCfCliNeverReads)
}
