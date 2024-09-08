package igo

import (
	"fmt"
	"io"
)

type kittyChunk struct {
	writer    io.Writer
	chunkSize int
}

// signal last chunk to Kitty on close
func (c *kittyChunk) Close() error {
	_, err := fmt.Fprint(c.writer, KITTY_IMG_HDR, "m=0;", KITTY_IMG_FTR)
	return err
}

func (c *kittyChunk) Write(buf []byte) (int, error) {
	bytesLeft := len(buf)
	written := 0

	for bytesLeft > 0 {

		toWrite := c.chunkSize
		if toWrite > bytesLeft {
			toWrite = bytesLeft
		}

		// prefix
		if _, err := fmt.Fprint(c.writer, KITTY_IMG_HDR, "m=1;"); err != nil {
			return written, err
		}

		// data
		n, err := c.writer.Write(buf[:toWrite])
		written += n
		if err != nil {
			return written, err
		}

		// suffix
		if _, err := fmt.Fprint(c.writer, KITTY_IMG_FTR); err != nil {
			return written, err
		}

		buf = buf[toWrite:]
		bytesLeft -= toWrite
	}

	return written, nil
}
