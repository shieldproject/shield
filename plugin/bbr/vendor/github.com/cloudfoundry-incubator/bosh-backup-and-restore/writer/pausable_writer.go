package writer

import (
	"io"
	"sync"
)

type PausableWriter struct {
	out            io.Writer
	mux            sync.Mutex
	paused         bool
	bufferedOutput []byte
}

func NewPausableWriter(out io.Writer) *PausableWriter {
	return &PausableWriter{out: out, paused: false}
}

func (pw *PausableWriter) Write(p []byte) (int, error) {
	pw.mux.Lock()
	defer pw.mux.Unlock()
	if pw.paused {
		pw.bufferedOutput = append(pw.bufferedOutput, p...)
		return 0, nil
	}
	return pw.out.Write(p)
}

func (pw *PausableWriter) Pause() {
	pw.mux.Lock()
	defer pw.mux.Unlock()
	pw.paused = true
}

func (pw *PausableWriter) Resume() (int, error) {
	pw.mux.Lock()
	defer pw.mux.Unlock()

	pw.paused = false
	n, err := pw.out.Write(pw.bufferedOutput)
	pw.bufferedOutput = []byte{}
	return n, err
}
