package udevdevice

type UdevDevice interface {
	KickDevice(filePath string)
	Settle() (err error)
	Trigger() (err error)
	EnsureDeviceReadable(filePath string) (err error)
}
