// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

// Package bufpipe implements a buffered pipe.
package bufpipe

import "io"
import "sync"

// There are a number of modes of operation that BufferPipe can operate in.
//
// As such, there are 4 different (and mostly orthogonal) flags that can be
// bitwise ORed together to create the mode of operation. Thus, with 4 flags,
// there are technically 16 different possible combinations (although, some of
// them are illogical). All 16 combinations are allowed, even if no sensible
// programmer should be using them.
//
// The first flag determines the buffer's structure (linear vs. ring). In linear
// mode, a writer can only write up to the internal buffer length's worth of
// data. On the other hand, in ring mode, the buffer is treated like a circular
// buffer and allow indefinite reading and writing.
//
// The second flag determines concurrent access to the pipe (mono vs. dual).
// In mono access mode, the writer has sole access to the pipe. Only after the
// Close method is called can a reader start reading data. In dual access
// mode, readers can read written data the moment it is ready.
//
// The third and fourth flag determines waiting protocol for reading and writing
// (polling vs. blocking). In blocking mode, a writer or reader will block until
// there is available buffer space or valid data to consume. In polling mode,
// read and write operations return immediately, possibly with an ErrShortWrite
// or ErrNoProgress error.
//
// Combining Ring with Mono is an illogical combination. Mono access dictates
// that no reader can drain the pipe until it is closed. However, the benefit
// of a Ring buffer is that it can circularly write data as a reader consumes
// the input. Ring with Mono is effectively Line with Mono.
//
// Combining Line with BlockI is an illogical combination. In Line mode, the
// amount that can be written is fixed and independent of how much is read.
// Thus, using BlockI in this case will cause the writer to block forever when
// the buffer is full.
//
// With all illogical combinations removed, there are only 8 logical
// combinations that programmers should use.
const (
	Ring   = 1 << iota // Ring buffer vs. linear buffer
	Dual               // Dual access IO vs. mono access IO
	BlockI             // Blocking input vs. polling input
	BlockO             // Blocking output vs. polling output

	// The below flags are the inverse of the ones above. They exist to make it
	// obvious what the inverse is.
	Line  = 0 // Inverse of Ring
	Mono  = 0 // Inverse of Dual
	PollI = 0 // Inverse of BlockI
	PollO = 0 // Inverse of BlockO
)

// The most common combination of flags are predefined with convenient aliases.
const (
	LineMono  = Line | Mono | PollI | BlockO
	LineDual  = Line | Dual | PollI | BlockO
	RingPoll  = Ring | Dual | PollI | PollO
	RingBlock = Ring | Dual | BlockI | BlockO
)

type BufferPipe struct {
	buf    []byte
	mode   int
	rdPtr  int64
	wrPtr  int64
	closed bool
	err    error
	mutex  sync.Mutex
	rdCond sync.Cond
	wrCond sync.Cond
}

// BufferPipe is similar in operation to io.Pipe and is intended to be the
// communication channel between producer and consumer routines. There are some
// specific use cases for BufferPipes over io.Pipe.
//
// First, in cases where a writer may need to go back a modify a portion of the
// past "written" data before allowing the reader to consume it.
// Second, for performance applications, where the cost of copying of data is
// noticeable. Thus, it would be more efficient to read/write data from/to the
// internal buffer directly.
//
// See the defined constants for more on the buffer mode of operation.
func NewBufferPipe(buf []byte, mode int) *BufferPipe {
	b := new(BufferPipe)
	b.buf = buf
	b.mode = mode
	b.rdCond.L = &b.mutex
	b.wrCond.L = &b.mutex
	return b
}

// The entire internal buffer. Be careful when touching the raw buffer.
// Line buffers are always guaranteed to be aligned to be front of the slice.
// Ring buffers use wrap around logic and could be physically split apart.
func (b *BufferPipe) Buffer() []byte {
	return b.buf
}

// The BufferPipe mode of operation.
func (b *BufferPipe) Mode() int {
	return b.mode
}

// The internal pointer values.
func (b *BufferPipe) Pointers() (rdPtr, wrPtr int64) {
	return b.rdPtr, b.wrPtr
}

// The total number of bytes the buffer can store.
func (b *BufferPipe) Capacity() int {
	return len(b.buf)
}

// The number of valid bytes that can be read.
func (b *BufferPipe) Length() int {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return int(b.wrPtr - b.rdPtr)
}

func (b *BufferPipe) writeWait() int {
	var rdZero int64 // Zero value
	isLine := b.mode&Ring == 0
	isBlock := b.mode&BlockI > 0

	rdPtr := &b.rdPtr
	if isLine {
		rdPtr = &rdZero // Amount read has no effect on amount available
	}
	if isBlock {
		for !b.closed && len(b.buf) == int(b.wrPtr-(*rdPtr)) {
			b.wrCond.Wait()
		}
	}
	if b.closed {
		return 0 // Closed buffer is never available
	}
	return len(b.buf) - int(b.wrPtr-(*rdPtr))
}

// Slices of available buffer that can be written to. This does not advance the
// internal write pointer. All of the available write space is the logical
// concatenation of the two slices.
//
// In linear buffers, the first slice obtained is guaranteed to be the entire
// available writable buffer slice.
//
// In LineMono mode only, slices obtained may still be modified even after
// WriteMark has been called and before Close. This is valid because this mode
// blocks readers until the buffer has been closed.
//
// In ring buffers, the first slice obtained may not represent all of the
// available buffer space since slices always represent a contiguous pieces of
// memory. However, the first slice is guaranteed to be a non-empty slice if
// space is available.
//
// In the Block mode, this method blocks until there is available space in
// the buffer to write. Poll mode, on the contrary, will return empty slices if
// the buffer is full.
func (b *BufferPipe) WriteSlices() (bufLo, bufHi []byte, err error) {
	if b == nil {
		return nil, nil, nil
	}
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.writeSlices()
}

func (b *BufferPipe) writeSlices() (bufLo, bufHi []byte, err error) {
	availCnt := b.writeWait() // Block until there is available buffer
	offLo := 0
	if len(b.buf) > 0 { // Prevent division by zero
		offLo = int(b.wrPtr) % len(b.buf)
	}
	offHi := offLo + availCnt
	if modCnt := offHi - len(b.buf); modCnt > 0 {
		offHi = len(b.buf)
		bufHi = b.buf[:modCnt] // Upper half (possible for Ring)
	}
	bufLo = b.buf[offLo:offHi] // Bottom half (will contain all data for Line)

	// Restrict the capacity to prevent users from accidentally going past end.
	bufLo = bufLo[:len(bufLo):len(bufLo)]
	bufHi = bufHi[:len(bufHi):len(bufHi)]

	// Check error status
	if len(bufLo) == 0 {
		switch {
		case b.err != nil:
			err = b.err
		case b.closed:
			err = io.ErrClosedPipe
		default:
			err = io.ErrShortWrite
		}
	}
	return
}

// Advances the write pointer.
//
// The amount that can be advanced must be non-negative and be less than the
// length of the slices returned by the previous WriteSlices. Calls to Write
// may not be done between these two calls. Also, another call to WriteMark is
// invalid until WriteSlices has been called again.
//
// If WriteMark is being used, only one writer routine is allowed.
func (b *BufferPipe) WriteMark(cnt int) {
	if b == nil && cnt == 0 {
		return
	}
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.writeMark(cnt)
}

func (b *BufferPipe) writeMark(cnt int) {
	availCnt := b.writeWait()
	if cnt < 0 || cnt > availCnt {
		panic("invalid mark increment value")
	}
	b.wrPtr += int64(cnt)

	b.rdCond.Signal()
}

// Write data to the buffer.
//
// In linear buffers, the length of the data slice may not exceed the capacity
// of the buffer. Otherwise, an ErrShortWrite error will be returned.
//
// In ring buffers, the length of the data slice may exceed the capacity.
//
// Under Block mode, this operation will block until all data has been written.
// If there is no consumer of the data, then this method may block forever.
func (b *BufferPipe) Write(data []byte) (cnt int, err error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	for cnt < len(data) {
		buf, _, err := b.writeSlices()
		if err != nil {
			return cnt, err
		}

		copyCnt := copy(buf, data[cnt:])
		b.writeMark(copyCnt)
		cnt += copyCnt
	}
	return cnt, nil
}

// Continually read the contents of the reader and write them to the pipe.
func (b *BufferPipe) ReadFrom(rd io.Reader) (cnt int64, err error) {
	for {
		b.mutex.Lock()
		buf, _, wrErr := b.writeSlices()
		rdPtr, rdErr := rd.Read(buf)
		b.writeMark(rdPtr)
		b.mutex.Unlock()
		cnt += int64(rdPtr)

		switch {
		case wrErr != nil:
			return cnt, wrErr
		case rdErr == io.EOF:
			return cnt, nil
		case rdErr != nil:
			return cnt, rdErr
		}
	}
}

func (b *BufferPipe) readWait() int {
	isBlock := b.mode&BlockO > 0
	isMono := b.mode&Dual == 0
	if isBlock {
		for !b.closed && b.rdPtr == b.wrPtr {
			b.rdCond.Wait()
		}
		for isMono && !b.closed {
			b.rdCond.Wait()
		}
	}
	if isMono && !b.closed {
		return 0
	}
	return int(b.wrPtr - b.rdPtr)
}

// Slices of valid data that can be read. This does not advance the internal
// read pointer. All of the valid readable data is the logical concatenation of
// the two slices.
//
// In linear buffers, the first slice obtained is guaranteed to be the entire
// valid readable buffer slice.
//
// In ring buffers, the first slice obtained may not represent all of the
// valid buffer data since slices always represent a contiguous pieces of
// memory. However, the first slice is guaranteed to be a non-empty slice if
// space is available.
//
// Under the Block mode, this method blocks until there is at least some valid
// data to read. The Mono mode is special in that, none of the data is
// considered ready for reading until the writer closes the channel.
func (b *BufferPipe) ReadSlices() (bufLo, bufHi []byte, err error) {
	if b == nil {
		return nil, nil, nil
	}
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.readSlices()
}

func (b *BufferPipe) readSlices() (bufLo, bufHi []byte, err error) {
	validCnt := b.readWait() // Block until there is valid buffer
	offLo := 0
	if len(b.buf) > 0 { // Prevent division by zero
		offLo = int(b.rdPtr) % len(b.buf)
	}
	offHi := offLo + validCnt
	if modCnt := offHi - len(b.buf); modCnt > 0 {
		offHi = len(b.buf)
		bufHi = b.buf[:modCnt] // Upper half (possible for Ring)
	}
	bufLo = b.buf[offLo:offHi] // Bottom half (will contain all data for Line)

	// Restrict the capacity to prevent users from accidentally going past end.
	bufLo = bufLo[:len(bufLo):len(bufLo)]
	bufHi = bufHi[:len(bufHi):len(bufHi)]

	// Check error status
	if len(bufLo) == 0 {
		switch {
		case b.err != nil:
			err = b.err
		case b.closed:
			err = io.EOF
		default:
			err = io.ErrNoProgress
		}
	}
	return
}

// Advances the read pointer.
//
// The amount that can be advanced must be non-negative and be less than the
// length of the slices returned by the previous ReadSlices. Calls to Read
// may not be done between these two calls. Also, another call to ReadMark is
// invalid until ReadSlices has been called again.
//
// If ReadMark is being used, only one reader routine is allowed.
func (b *BufferPipe) ReadMark(cnt int) {
	if b == nil && cnt == 0 {
		return
	}
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.readMark(cnt)
}

func (b *BufferPipe) readMark(cnt int) {
	validCnt := b.readWait()
	if cnt < 0 || cnt > validCnt {
		panic("invalid mark increment value")
	}
	b.rdPtr += int64(cnt)

	b.wrCond.Signal()
}

// Read data from the buffer.
//
// In all modes, the length of the data slice may exceed the capacity of
// the buffer. The operation will block until all data has been read or until
// the EOF is hit.
//
// Under Block mode, this method may block forever if there is no producer.
func (b *BufferPipe) Read(data []byte) (cnt int, err error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	for cnt < len(data) {
		buf, _, err := b.readSlices()
		if err != nil {
			return cnt, err
		}

		copyCnt := copy(data[cnt:], buf)
		b.readMark(copyCnt)
		cnt += copyCnt
	}
	return cnt, nil
}

// Continually read the contents of the pipe and write them to the writer.
func (b *BufferPipe) WriteTo(wr io.Writer) (cnt int64, err error) {
	for {
		b.mutex.Lock()
		data, _, rdErr := b.readSlices()
		wrPtr, wrErr := wr.Write(data)
		b.readMark(wrPtr)
		b.mutex.Unlock()
		cnt += int64(wrPtr)

		switch {
		case wrErr != nil:
			return cnt, wrErr
		case rdErr == io.EOF:
			return cnt, nil
		case rdErr != nil:
			return cnt, rdErr
		}
	}
}

// Close the buffer down.
//
// All write operations have no effect after this, while all read operations
// will drain remaining data in the buffer. This operation is somewhat similar
// to how Go channels operate.
//
// Writers should close the buffer to indicate to readers to mark end-of-stream.
//
// Readers should only close the buffer in the event of unexpected termination.
// The mechanism allows readers to inform writers of consumer termination and
// prevents the producer from potentially being blocked forever.
func (b *BufferPipe) Close() error {
	return b.CloseWithError(nil)
}

// Closes the pipe with the given error. This sets the error value for the pipe
// and returns the previous error value.
func (b *BufferPipe) CloseWithError(err error) (errPre error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	errPre, b.err = b.err, err
	b.closed = true
	b.rdCond.Broadcast()
	b.wrCond.Broadcast()
	return errPre
}

// Roll back the write pointer and return the number of bytes rolled back.
// If successful, this effectively makes the valid length zero. In order to
// prevent race conditions with the reader, this action is only valid in Mono
// access mode before the channel is closed.
func (b *BufferPipe) Rollback() int {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if b.closed || b.mode&Dual > 0 {
		return 0
	}
	cnt := b.wrPtr - b.rdPtr
	b.wrPtr = b.rdPtr
	return int(cnt)
}

// Makes the buffer ready for use again by opening the pipe for writing again.
// The read and write pointers will be reset to zero and errors will be cleared.
func (b *BufferPipe) Reset() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.wrPtr, b.rdPtr = 0, 0
	b.err, b.closed = nil, false
}

// TODO(jtsai): Allow BufferPipe to be grown. This is safe at Reset time and
// before the first call to Write. When else is this safe?

// TODO(jtsai): Double check why some methods allow the BufferPipe pointer to
// be nil.

// TODO(jtsai): Why are some methods not protected by a mutex?
