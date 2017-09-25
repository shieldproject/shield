// +build windows

package monitor

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
	"unsafe"
)

// https://msdn.microsoft.com/en-us/library/windows/desktop/aa366589(v=vs.85).aspx
var procGlobalMemoryStatusEx = kernel32DLL.MustFindProc("GlobalMemoryStatusEx")

type Byte uint64

const (
	KB Byte = 1 << (10 * (iota + 1))
	MB
	GB
)

func (b Byte) Uint64() uint64 { return uint64(b) }

func (b Byte) String() string {
	switch {
	case b < KB:
		return fmt.Sprintf("%d", b)
	case b < MB:
		return fmt.Sprintf("%.1fK", float64(b)/float64(KB))
	case b < GB:
		return fmt.Sprintf("%.1fM", float64(b)/float64(MB))
	}
	return fmt.Sprintf("%.1fG", float64(b)/float64(GB))
}

type MemStat struct {
	Total Byte
	Avail Byte
}

func (m MemStat) Used() float64 {
	if m.Avail == 0 {
		if m.Total == 0 {
			return 0
		}
		return 1
	}
	return 1 - float64(m.Avail)/float64(m.Total)
}

type memorystatusex struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

func getGlobalMemoryStatusEx() (*memorystatusex, error) {
	var m memorystatusex
	m.Length = uint32(unsafe.Sizeof(m))
	r1, _, e1 := syscall.Syscall(procGlobalMemoryStatusEx.Addr(), 1, uintptr(unsafe.Pointer(&m)), 0, 0)
	if err := checkErrno(r1, e1); err != nil {
		return nil, fmt.Errorf("GlobalMemoryStatusEx: %s", err)
	}
	return &m, nil
}

func SystemMemStats() (MemStat, error) {
	m, err := getGlobalMemoryStatusEx()
	if err != nil {
		return MemStat{}, err
	}
	mem := MemStat{
		Total: Byte(m.TotalPhys),
		Avail: Byte(m.AvailPhys),
	}
	return mem, nil
}

func SystemPageStats() (MemStat, error) {
	const MB = 1024 * 1024
	out, err := exec.Command("wmic", "pagefile", "list", "full").Output()
	if err != nil {
		return MemStat{}, err
	}
	total, err := parseWmicOutput(out, []byte("AllocatedBaseSize"))
	if err != nil {
		return MemStat{}, err
	}
	used, err := parseWmicOutput(out, []byte("CurrentUsage"))
	if err != nil {
		return MemStat{}, err
	}
	total *= MB
	used *= MB
	mem := MemStat{
		Total: Byte(total),
		Avail: Byte(total - used),
	}
	return mem, nil
}

func parseWmicOutput(s, sep []byte) (uint64, error) {
	bb := bytes.Split(s, []byte("\n"))
	for i := 0; i < len(bb); i++ {
		b := bytes.TrimSpace(bb[i])
		if bytes.HasPrefix(b, sep) {
			n := bytes.IndexByte(b, '=')
			if n == -1 || n == len(s)-1 {
				return 0, errors.New("parseWmicOutput: parsing field: " + string(sep))
			}
			return strconv.ParseUint(string(b[n+1:]), 10, 64)
		}
	}
	return 0, errors.New("parseWmicOutput: missing field: " + string(sep))
}
