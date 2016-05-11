package pivnet

import (
	"fmt"
	"io"
	"math"
	"strings"
)

type DownloadReader struct {
	io.ReadCloser
	accumulatedLength int64
	Writer            io.Writer
	ContentLength     int64
}

func (dr *DownloadReader) Read(p []byte) (int, error) {
	length, err := dr.ReadCloser.Read(p)
	dr.accumulatedLength += int64(length)

	if err == nil {
		dr.displayProgress(dr.accumulatedLength)
	}

	return length, err
}

func (dr *DownloadReader) displayProgress(length int64) {
	percentage := float64(length) / float64(dr.ContentLength)
	bars := int(math.Ceil(20 * percentage))

	fmt.Fprintf(dr.Writer,
		"\rProgress: |%s>%s| %d%%",
		strings.Repeat("=", bars),
		strings.Repeat(" ", 20-bars),
		int(math.Ceil(percentage*100)))
}
