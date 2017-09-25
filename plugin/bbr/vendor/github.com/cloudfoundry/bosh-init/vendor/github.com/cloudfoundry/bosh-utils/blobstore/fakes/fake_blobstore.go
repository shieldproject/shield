package fakes

type FakeBlobstore struct {
	GetBlobIDs      []string
	GetFingerprints []string
	GetFileName     string
	GetFileNames    []string
	GetError        error
	GetErrs         []error

	CleanUpFileName string
	CleanUpErr      error

	DeleteBlobID string
	DeleteErr    error

	CreateFileNames    []string
	CreateBlobID       string
	CreateBlobIDs      []string
	CreateFingerprint  string
	CreateFingerprints []string
	CreateErr          error
	CreateErrs         []error
	CreateCallBack     func()

	ValidateError error
}

func NewFakeBlobstore() *FakeBlobstore {
	return &FakeBlobstore{}
}

func (bs *FakeBlobstore) Get(blobID, fingerprint string) (string, error) {
	bs.GetBlobIDs = append(bs.GetBlobIDs, blobID)
	bs.GetFingerprints = append(bs.GetFingerprints, fingerprint)

	fileName, err := bs.GetFileName, bs.GetError

	if len(bs.GetFileNames) > 0 {
		fileName = bs.GetFileNames[0]
		bs.GetFileNames = bs.GetFileNames[1:]
	}

	if len(bs.GetErrs) > 0 {
		err = bs.GetErrs[0]
		bs.GetErrs = bs.GetErrs[1:]
	}

	return fileName, err
}

func (bs *FakeBlobstore) CleanUp(fileName string) error {
	bs.CleanUpFileName = fileName
	return bs.CleanUpErr
}

func (bs *FakeBlobstore) Delete(blobId string) error {
	bs.DeleteBlobID = blobId
	return bs.DeleteErr
}

func (bs *FakeBlobstore) Create(fileName string) (string, string, error) {
	bs.CreateFileNames = append(bs.CreateFileNames, fileName)

	if bs.CreateCallBack != nil {
		bs.CreateCallBack()
	}

	blobID, fingerprint, err := bs.CreateBlobID, bs.CreateFingerprint, bs.CreateErr

	if len(bs.CreateBlobIDs) > 0 {
		blobID = bs.CreateBlobIDs[0]
		bs.CreateBlobIDs = bs.CreateBlobIDs[1:]
	}

	if len(bs.CreateFingerprints) > 0 {
		fingerprint = bs.CreateFingerprints[0]
		bs.CreateFingerprints = bs.CreateFingerprints[1:]
	}

	if len(bs.CreateErrs) > 0 {
		err = bs.CreateErrs[0]
		bs.CreateErrs = bs.CreateErrs[1:]
	}

	return blobID, fingerprint, err
}

func (bs *FakeBlobstore) Validate() error {
	return bs.ValidateError
}
