package deviceutil

type DeviceUtil interface {
	GetFilesContents(fileNames []string) (contents [][]byte, err error)
	GetBlockDeviceSize() (size uint64, err error)
}
