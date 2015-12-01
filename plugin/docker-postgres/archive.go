package main

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

type ArchiveReader struct {
	reader *tar.Reader
}

func NewArchiveReader(io io.Reader) *ArchiveReader {
	return &ArchiveReader{reader: tar.NewReader(io)}
}

func (a *ArchiveReader) Next(metadata interface{}) (io.Reader, error) {
	header, err := a.reader.Next()
	if err == io.EOF {
		return nil, err
	}

	raw := make([]byte, header.Size)
	_, err = a.reader.Read(raw)
	if err != nil {
		return nil, err
	}

	header, err = a.reader.Next()
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(raw, &metadata)
	if err != nil {
		return nil, err
	}

	return a.reader, nil
}

type ArchiveWriter struct {
	writer *tar.Writer
}

func NewArchiveWriter(io io.Writer) *ArchiveWriter {
	return &ArchiveWriter{writer: tar.NewWriter(io)}
}

func (a *ArchiveWriter) Write(prefix string, metadata interface{}, data *os.File) error {
	// JSONify the metadata
	raw, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	// how much data is there?
	size, err := data.Seek(0, 1)
	if err != nil {
		return err
	}

	// rewind
	_, err = data.Seek(0, 0)
	if err != nil {
		return err
	}

	a.writer.WriteHeader(&tar.Header{
		Name:    fmt.Sprintf("%s.meta", prefix),
		Mode:    0644,
		Size:    int64(len(raw)),
		ModTime: time.Now(),
	})
	a.writer.Write(raw)

	a.writer.WriteHeader(&tar.Header{
		Name:    fmt.Sprintf("%s.data", prefix),
		Mode:    0644,
		Size:    int64(size),
		ModTime: time.Now(),
	})
	buf := make([]byte, 8192)
	for {
		n, err := data.Read(buf)
		if n > 0 {
			a.writer.Write(buf[0:n])
		}
		if err != nil {
			break
		}
	}

	a.writer.Flush()
	return nil
}

func (a *ArchiveWriter) Close() {
	a.writer.Close()
}
