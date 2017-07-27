package release

type BuiltReader struct {
	releaseReader Reader
	devIndicies   ArchiveIndicies
	finalIndicies ArchiveIndicies
}

func NewBuiltReader(
	releaseReader Reader,
	devIndicies ArchiveIndicies,
	finalIndicies ArchiveIndicies,
) BuiltReader {
	return BuiltReader{
		releaseReader: releaseReader,
		devIndicies:   devIndicies,
		finalIndicies: finalIndicies,
	}
}

func (r BuiltReader) Read(path string) (Release, error) {
	release, err := r.releaseReader.Read(path)
	if err != nil {
		return nil, err
	}

	err = release.Build(r.devIndicies, r.finalIndicies)
	if err != nil {
		return nil, err
	}

	return release, nil
}
