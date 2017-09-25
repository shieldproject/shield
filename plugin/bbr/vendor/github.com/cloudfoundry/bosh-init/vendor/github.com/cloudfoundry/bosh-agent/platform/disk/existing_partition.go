package disk

type existingPartition struct {
	Index        int
	SizeInBytes  uint64
	StartInBytes uint64
	EndInBytes   uint64
}
