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
	lastPercentage    int
	Writer            io.Writer
	ContentLength     int64
	ExistingLength    int64
}

func (dr *DownloadReader) Read(p []byte) (int, error) {
	dr.displayProgress(dr.accumulatedLength)

	length, err := dr.ReadCloser.Read(p)
	dr.accumulatedLength += int64(length)

	if err == nil {
		dr.displayProgress(dr.accumulatedLength)
	}

	return length, err
}

func (dr *DownloadReader) displayProgress(length int64) {
	totalLength := float64(dr.ExistingLength + dr.ContentLength)
	totalPercentage := float64(dr.ExistingLength+length) / totalLength
	downloadedPercentage := float64(length) / totalLength
	existingPercentage := float64(dr.ExistingLength) / totalLength

	plusses := int(math.Ceil(20 * existingPercentage))
	bars := int(math.Ceil(20 * downloadedPercentage))
	bars = int(math.Min(float64(bars), float64(20-plusses)))
	spaces := 20 - bars - plusses
	percentage := int(math.Ceil(totalPercentage * 100))

	if percentage != 0 && dr.lastPercentage == percentage {
		return
	}
	dr.lastPercentage = percentage

	fmt.Fprintf(dr.Writer,
		"\rProgress: |%s%s>%s| %d%% ",
		strings.Repeat("+", plusses),
		strings.Repeat("=", bars),
		strings.Repeat(" ", spaces),
		percentage)
}
