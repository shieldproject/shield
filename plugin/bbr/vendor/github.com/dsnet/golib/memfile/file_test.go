// Copyright 2017, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package memfile

import (
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestFile(t *testing.T) {
	type (
		testRead struct {
			cnt      int
			wantData string
			wantErr  error
		}
		testReadAt struct {
			cnt      int
			pos      int64
			wantData string
			wantErr  error
		}
		testWrite struct {
			data    string
			wantCnt int
			wantErr error
		}
		testWriteAt struct {
			data    string
			pos     int64
			wantCnt int
			wantErr error
		}
		testSeek struct {
			offset  int64
			whence  int
			wantPos int64
			wantErr error
		}
		testTruncate struct {
			size    int64
			wantErr error
		}
		testBytes struct {
			wantData string
		}
	)

	sz := func(n int) string { return strings.Repeat("\x00", n) }
	tests := []interface{}{ // []T where T is one of the testX types above
		testRead{10, "", io.EOF},
		testSeek{5, io.SeekEnd, 5, nil},
		testSeek{5, io.SeekEnd, 5, nil},
		testWrite{"abcdefghijklmnopqrstuvwxyz", 26, nil},
		testTruncate{25, nil},
		testBytes{sz(5) + "abcdefghijklmnopqrst"},
		testTruncate{-1, errInvalid},
		testTruncate{10, nil},
		testWriteAt{"ABCDE", 15, 5, nil},
		testBytes{sz(5) + "abcde" + sz(5) + "ABCDE"},
		testReadAt{10, 15, "ABCDE", io.EOF},
		testReadAt{10, -15, "", errInvalid},
		testWriteAt{"ABCDE", -15, 0, errInvalid},
		testReadAt{8, 3, sz(2) + "abcde" + sz(1), nil},
		testSeek{0, io.SeekCurrent, 31, nil},
		testSeek{-32, io.SeekCurrent, 0, errInvalid},
		testSeek{-11, io.SeekCurrent, 20, nil},
		testWrite{"#", 1, nil},
		testBytes{sz(5) + "abcde" + sz(5) + "ABCDE#"},
		testTruncate{0, nil},
		testWrite{"#", 1, nil},
		testBytes{sz(21) + "#"},
		testSeek{5, io.SeekStart, 5, nil},
		testWrite{"12345", 5, nil},
		testSeek{5, io.SeekStart, 5, nil},
		testRead{5, "12345", nil},
		testSeek{-23, io.SeekEnd, 0, errInvalid},
		testSeek{0, io.SeekEnd, 22, nil},
		testSeek{-22, io.SeekEnd, 0, nil},
		testRead{10, sz(5) + "12345", nil},
		testSeek{0, io.SeekCurrent, 10, nil},
		testTruncate{0, nil},
		testBytes{""},
	}

	// Create a new File that emulates a file's operations.
	fb := new(File)

	// Open a temporary file to match behavior with.
	ft, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer ft.Close()
	defer os.Remove(ft.Name())

	for i, tt := range tests {
		switch tt := tt.(type) {
		case testRead:
			b := make([]byte, tt.cnt)

			n, gotErr := readFull(fb, b)
			gotData := string(b[:n])
			if gotData != tt.wantData || gotErr != tt.wantErr {
				t.Fatalf("test %d, Read():\ngot  (%q, %v)\nwant (%q, %v)", i, gotData, gotErr, tt.wantData, tt.wantErr)
			}

			n, wantErr := readFull(ft, b)
			wantData := string(b[:n])
			if gotData != wantData || (gotErr == nil) != (wantErr == nil) {
				t.Fatalf("test %d, Read():\ngot  (%q, %v)\nwant (%q, %v)", i, gotData, gotErr, wantData, wantErr)
			}
		case testReadAt:
			b := make([]byte, tt.cnt)

			n, gotErr := fb.ReadAt(b, tt.pos)
			gotData := string(b[:n])
			if gotData != tt.wantData || gotErr != tt.wantErr {
				t.Fatalf("test %d, ReadAt():\ngot  (%q, %v)\nwant (%q, %v)", i, gotData, gotErr, tt.wantData, tt.wantErr)
			}

			n, wantErr := ft.ReadAt(b, tt.pos)
			wantData := string(b[:n])
			if gotData != wantData || (gotErr == nil) != (wantErr == nil) {
				t.Fatalf("test %d, ReadAt():\ngot  (%q, %v)\nwant (%q, %v)", i, gotData, gotErr, wantData, wantErr)
			}
		case testWrite:
			gotCnt, gotErr := fb.Write([]byte(tt.data))
			if gotCnt != tt.wantCnt || gotErr != tt.wantErr {
				t.Fatalf("test %d, Write():\ngot  (%d, %v)\nwant (%d, %v)", i, gotCnt, gotErr, tt.wantCnt, tt.wantErr)
			}

			wantCnt, wantErr := ft.Write([]byte(tt.data))
			if gotCnt != wantCnt || (gotErr == nil) != (wantErr == nil) {
				t.Fatalf("test %d, Write():\ngot  (%d, %v)\nwant (%d, %v)", i, gotCnt, gotErr, wantCnt, wantErr)
			}
		case testWriteAt:
			gotCnt, gotErr := fb.WriteAt([]byte(tt.data), tt.pos)
			if gotCnt != tt.wantCnt || gotErr != tt.wantErr {
				t.Fatalf("test %d, WriteAt():\ngot  (%d, %v)\nwant (%d, %v)", i, gotCnt, gotErr, tt.wantCnt, tt.wantErr)
			}

			wantCnt, wantErr := ft.WriteAt([]byte(tt.data), tt.pos)
			if gotCnt != wantCnt || (gotErr == nil) != (wantErr == nil) {
				t.Fatalf("test %d, WriteAt():\ngot  (%d, %v)\nwant (%d, %v)", i, gotCnt, gotErr, wantCnt, wantErr)
			}
		case testSeek:
			gotPos, gotErr := fb.Seek(tt.offset, tt.whence)
			if gotPos != tt.wantPos || gotErr != tt.wantErr {
				t.Fatalf("test %d, Seek():\ngot  (%d, %v)\nwant (%d, %v)", i, gotPos, gotErr, tt.wantPos, tt.wantErr)
			}

			wantPos, wantErr := ft.Seek(tt.offset, tt.whence)
			if gotPos != wantPos || (gotErr == nil) != (wantErr == nil) {
				t.Fatalf("test %d, Seek():\ngot  (%d, %v)\nwant (%d, %v)", i, gotPos, gotErr, wantPos, wantErr)
			}
		case testTruncate:
			gotErr := fb.Truncate(tt.size)
			if gotErr != tt.wantErr {
				t.Fatalf("test %d, Truncate() = %v, want %v", i, gotErr, tt.wantErr)
			}

			wantErr := ft.Truncate(tt.size)
			if (gotErr == nil) != (wantErr == nil) {
				t.Fatalf("test %d, Truncate() = %v, want %v", i, gotErr, wantErr)
			}
		case testBytes:
			gotData := string(fb.Bytes())
			if gotData != tt.wantData {
				t.Fatalf("test %d, Bytes():\ngot  %q\nwant %q", i, gotData, tt.wantData)
			}

			wantData, err := ioutil.ReadFile(ft.Name())
			if err != nil {
				t.Fatalf("test %d, unexpected ReadFile error: %v", i, err)
			}
			if gotData != string(wantData) {
				t.Fatalf("test %d, Bytes():\ngot  %q\nwant %q", i, gotData, wantData)
			}
		default:
			t.Fatalf("test %d, unknown test operation: %T", i, tt)
		}
	}
}

func readFull(r io.Reader, b []byte) (n int, err error) {
	b0 := b
	for len(b) > 0 && err == nil {
		n, err = r.Read(b)
		b = b[n:]
	}
	if len(b) == 0 && err == io.EOF {
		err = nil
	}
	return len(b0) - len(b), err
}
