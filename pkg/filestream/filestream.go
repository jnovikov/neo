package filestream

import (
	"errors"
	"fmt"
	"io"

	neopb "neo/lib/genproto/neo"
)

const (
	chunkSize = 4 * 1024 * 1024 // 2MB
)

type DownloadStream interface {
	Recv() (*neopb.FileStream, error)
}

type UploadStream interface {
	Send(*neopb.FileStream) error
}

func Save(stream DownloadStream, out io.Writer) error {
	for {
		in, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("reading from stream: %w", err)
		}
		if _, err := out.Write(in.GetChunk()); err != nil {
			return fmt.Errorf("writing stream content chunk: %w", err)
		}
	}
}

func Load(in io.Reader, stream UploadStream) error {
	b := make([]byte, chunkSize)
	for {
		n, err := in.Read(b)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("reading content: %w", err)
		}
		if err := stream.Send(&neopb.FileStream{Chunk: b[:n]}); err != nil {
			return fmt.Errorf("sending chunk to stream: %w", err)
		}
	}
}
