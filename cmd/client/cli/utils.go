package cli

import "bytes"

func isBinary(data []byte) bool {
	return bytes.Equal(data[:4], []byte("\x7fELF"))
}
