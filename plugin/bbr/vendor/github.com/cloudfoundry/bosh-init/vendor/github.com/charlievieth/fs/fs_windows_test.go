// +build windows

package fs

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func longPathName() string {
	var buf bytes.Buffer
	for i := 0; i < 2; i++ {
		for i := byte('A'); i <= 'Z'; i++ {
			buf.Write(bytes.Repeat([]byte{i}, 4))
			buf.WriteRune(filepath.Separator)
		}
	}
	return filepath.Clean(buf.String())
}

func TestLeadingSlash(t *testing.T) {
	s := "0123456789abcdef"
	s = s + s + s + s + s + s + s + s + s + s + s + s + s + s

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// [drive letter]:\path\to\testdata
	longpath := filepath.Join(dir, "testdata", s)

	// \path\to\testdata
	testpath := strings.TrimPrefix(longpath, filepath.VolumeName(longpath))

	f, err := Create(longpath)
	if err != nil {
		t.Fatalf("WTF: %s\n", err)
	}
	f.Close()
	defer os.RemoveAll(`\\?\` + longpath)

	// Test with leading backslash
	f, err = Open(testpath)
	if err != nil {
		t.Fatalf("TestLeadingSlash (%s): failed to open file (%s): %s\n", longpath, testpath, err)
	}
	f.Close()

	// Test with forward slashes
	unixpath := strings.Replace(testpath, `\`, `/`, -1)
	f, err = Open(unixpath)
	if err != nil {
		t.Fatalf("TestLeadingSlash (%s): failed to open file with forward slashes (%s): %s\n", longpath, unixpath, err)
	}
	f.Close()

	relative := "./testdata/" + s
	f, err = Open(relative)
	if err != nil {
		t.Fatalf("TestLeadingSlash (%s): failed to open relative path (%s): %s\n", longpath, relative, err)
	}
	f.Close()

	if err := Remove(longpath); err != nil {
		t.Fatalf("TestLeadingSlash (%s): failed to cleanup: %s\n", longpath, err)
	}
}

func TestRemoveAll(t *testing.T) {
	name := longPathName()
	temp := os.TempDir()
	path := filepath.Join(temp, name)
	target := filepath.Join(temp, strings.Split(name, string(os.PathSeparator))[0])

	err := MkdirAll(path, 0755)
	if err != nil {
		t.Fatalf("TestRemoveAll: %s", err)
	}
	defer os.RemoveAll(`\\?\` + target)

	// TODO: cleanup on failure
	if err := RemoveAll(target); err != nil {
		t.Fatalf("TestRemoveAll: %s\n", err)
	}
	if _, err := Stat(path); err == nil {
		t.Fatalf("TestRemoveAll: failed to remove directory: %s\n", path)
	}
	if _, err := Stat(target); err == nil {
		t.Fatalf("TestRemoveAll: failed to remove directory: %s\n", target)
	}
}

func TestMkdirAll(t *testing.T) {
	name := longPathName()
	temp := os.TempDir()
	path := filepath.Join(temp, name)
	target := filepath.Join(temp, strings.Split(name, string(os.PathSeparator))[0])

	err := MkdirAll(path, 0755)
	if err != nil {
		t.Fatalf("TestMkdirAll: %s", err)
	}
	defer os.RemoveAll(`\\?\` + target)

	if _, err := Stat(path); err != nil {
		t.Fatalf("TestMkdirAll: Stat failed %s\n", err)
	}
	// Make sure the handling of long paths is case-insensitive
	if _, err := Stat(strings.ToLower(path)); err != nil {
		t.Fatalf("TestMkdirAll: Stat failed %s\n", err)
	}
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("TestMkdirAll: RemoveAll %s\n", err)
	}
}
