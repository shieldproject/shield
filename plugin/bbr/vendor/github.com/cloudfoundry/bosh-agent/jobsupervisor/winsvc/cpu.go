// +build windows

package winsvc

import (
	"math"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	// Global kernel32 DLL
	kernel32DLL = windows.NewLazySystemDLL("kernel32")

	// https://msdn.microsoft.com/en-us/library/windows/desktop/ms724400(v=vs.85).aspx
	procGetSystemTimes = kernel32DLL.NewProc("GetSystemTimes")
)

var cpu = newMonitor(time.Second * 5)

type monitor struct {
	user   cpuTime
	kernel cpuTime
	idle   cpuTime
	mu     sync.RWMutex
	tick   *time.Ticker
}

func newMonitor(tick time.Duration) *monitor {
	if tick < time.Second {
		tick = time.Second
	}
	m := &monitor{
		tick: time.NewTicker(tick),
	}
	go m.monitorLoop()
	return m
}

func (m *monitor) CPU() (usage float64) {
	m.mu.RLock()
	usage = (m.user.load + m.kernel.load) * 100
	m.mu.RUnlock()
	return
}

func (m *monitor) Stop() error {
	if m.tick != nil {
		m.tick.Stop()
	}
	return nil
}

func (m *monitor) monitorLoop() {
	m.updateSystemCPU()
	for range m.tick.C {
		m.updateSystemCPU()
	}
}

func (m *monitor) updateSystemCPU() {
	if procGetSystemTimes.Find() != nil {
		return
	}
	var (
		idleTime   filetime
		kernelTime filetime
		userTime   filetime
	)
	r1, _, e1 := syscall.Syscall(procGetSystemTimes.Addr(), 3,
		uintptr(unsafe.Pointer(&idleTime)),
		uintptr(unsafe.Pointer(&kernelTime)),
		uintptr(unsafe.Pointer(&userTime)),
	)
	if r1 == 0 {
		_ = e1 // unused for now
		return
	}
	m.calculateSystemCPU(kernelTime.Uint64(), userTime.Uint64(), idleTime.Uint64())
}

func (m *monitor) calculateSystemCPU(kernelTicks, userTicks, idleTicks uint64) {
	m.mu.Lock()

	kernel := kernelTicks - m.kernel.previous
	user := userTicks - m.user.previous
	idle := idleTicks - m.idle.previous

	total := kernel + user
	if total > 0 {
		m.idle.load = float64(idle) / float64(total)
		m.idle.previous = idleTicks
		m.idle.delta = idle

		m.kernel.load = math.Max(float64(kernel-idle)/float64(total), 0)
		m.kernel.previous = kernelTicks
		m.kernel.delta = kernel

		m.user.load = math.Max(1-m.idle.load-m.kernel.load, 0)
		m.user.previous = userTicks
		m.user.delta = user
	} else {
		m.idle.load = 0
		m.kernel.load = 0
		m.user.load = 0
	}

	m.mu.Unlock()
}

type cpuTime struct {
	previous uint64
	delta    uint64
	load     float64
}

func (c cpuTime) CPU() float64 { return c.load }

// https://msdn.microsoft.com/en-us/library/windows/desktop/ms724284(v=vs.85).aspx
type filetime struct {
	LowDateTime  uint32
	HighDateTime uint32
}

func (f filetime) Uint64() uint64 {
	return uint64(f.HighDateTime)<<32 | uint64(f.LowDateTime)
}
