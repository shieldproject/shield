package fakes

type FakeDeviceUtil struct {
	GetFilesContentsFileNames []string
	GetFilesContentsError     error
	GetFilesContentsContents  [][]byte

	GetBlockDeviceSizeError error
	GetBlockDeviceSizeSize  uint64
}

func NewFakeDeviceUtil() (util *FakeDeviceUtil) {
	util = &FakeDeviceUtil{}
	return
}

func (util *FakeDeviceUtil) GetFilesContents(fileNames []string) ([][]byte, error) {
	util.GetFilesContentsFileNames = fileNames
	return util.GetFilesContentsContents, util.GetFilesContentsError
}

func (util *FakeDeviceUtil) GetBlockDeviceSize() (size uint64, err error) {
	return util.GetBlockDeviceSizeSize, util.GetBlockDeviceSizeError
}
