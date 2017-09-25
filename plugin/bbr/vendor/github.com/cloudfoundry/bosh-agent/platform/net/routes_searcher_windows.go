package net

import (
	"bytes"
	"errors"
	"net"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

var (
	modiphlpapi         = syscall.MustLoadDLL("iphlpapi.dll")
	procGetAdaptersInfo = modiphlpapi.MustFindProc("GetAdaptersInfo")
)

type windowsRoutesSearcher struct{}

func NewRoutesSearcher(_ boshsys.CmdRunner) RoutesSearcher {
	return windowsRoutesSearcher{}
}

func (s windowsRoutesSearcher) SearchRoutes() ([]Route, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, bosherr.WrapError(err, "Running route")
	}
	var routes []Route
	var first error
	for _, fs := range ifs {
		gateway, err := getGateway(fs.Index)
		if err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		route := Route{
			InterfaceName: fs.Name,
			Gateway:       gateway,
			Destination:   "0.0.0.0",
		}
		routes = append(routes, route)
	}
	if len(routes) == 0 {
		if first != nil {
			return nil, first
		}
		return nil, bosherr.Error("no routes")
	}
	return routes, nil
}

func toString(b []byte) string {
	n := bytes.IndexByte(b, 0)
	if n == -1 {
		n = len(b)
	}
	return string(b[:n])
}

func getGateway(index int) (string, error) {
	first, err := getAdaptersInfo()
	if err != nil {
		return "", err
	}
	for info := first; info != nil; info = info.Next {
		if int(info.Index) == index {
			addr := toString(info.GatewayList.IpAddress.String[0:])
			return addr, nil
		}
	}

	return "", errors.New("interface not found")
}

func getAdaptersInfo() (*windows.IpAdapterInfo, error) {
	for n := 4096; n < 65536; n *= 2 {
		bufLen := uint32(n)
		buf := make([]byte, n)
		r0, _, _ := syscall.Syscall(procGetAdaptersInfo.Addr(), 2,
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(unsafe.Pointer(&bufLen)),
			0,
		)
		switch syscall.Errno(r0) {
		case 0:
			return (*windows.IpAdapterInfo)(unsafe.Pointer(&buf[0])), nil
		case windows.ERROR_BUFFER_OVERFLOW:
			// continue
		default:
			return nil, syscall.Errno(r0)
		}
	}
	return nil, errors.New("insufficient allocation")
}
