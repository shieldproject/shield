// +build windows

package monitor

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	// https://msdn.microsoft.com/en-us/library/windows/desktop/aa364935(v=vs.85).aspx
	procGetDiskFreeSpace = kernel32DLL.MustFindProc("GetDiskFreeSpaceW")
)

type DiskUsage struct {
	Total Byte
	Used  Byte
}

func (d *DiskUsage) Percent() float64 {
	if d.Total > 0 {
		return float64(d.Used) / float64(d.Total)
	}
	return 0
}

func newDiskUsage(u diskUsage) DiskUsage {
	m := uint64(u.SectorsPerCluster * u.BytesPerSector)
	total := uint64(u.TotalNumberOfClusters) * m
	used := total - (uint64(u.NumberOfFreeClusters) * m)
	return DiskUsage{
		Total: Byte(total),
		Used:  Byte(used),
	}
}

type diskUsage struct {
	SectorsPerCluster     uint32
	BytesPerSector        uint32
	NumberOfFreeClusters  uint32
	TotalNumberOfClusters uint32
}

func UsedDiskSpace(name string) (DiskUsage, error) {
	u, err := getDiskFreeSpace(name)
	return newDiskUsage(u), err
}

func getDiskFreeSpace(name string) (diskUsage, error) {
	var u diskUsage
	root, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return u, fmt.Errorf("UsedDiskSpace (%s): %s", name, err)
	}
	r1, _, e1 := syscall.Syscall6(procGetDiskFreeSpace.Addr(), 5,
		uintptr(unsafe.Pointer(root)),
		uintptr(unsafe.Pointer(&u.SectorsPerCluster)),
		uintptr(unsafe.Pointer(&u.BytesPerSector)),
		uintptr(unsafe.Pointer(&u.NumberOfFreeClusters)),
		uintptr(unsafe.Pointer(&u.TotalNumberOfClusters)),
		0,
	)
	if err := checkErrno(r1, e1); err != nil {
		return u, fmt.Errorf("UsedDiskSpace (%s): %s", name, err)
	}
	return u, nil
}
