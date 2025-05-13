package feather

import "io"

// LimitedReader reads from R but limits the amount of data returned to just N bytes.
// Each call to Read updates N to reflect the new amount remaining.
// Read returns ErrLimitedReaderEOF when N <= 0 or when the underlying R returns EOF.
// Unlike the std io.LimitedReader this provides feedback that the limit was reached through the returned error.
type LimitedReader struct {
	R io.Reader
	N int64 // bytes allotted
}

// LimitReader returns a LimitedReader that reads from r but stops with ErrLimitedReaderEOF after n bytes.
func limitReader(r io.Reader, n int64) *LimitedReader {
	return &LimitedReader{R: r, N: n}
}
