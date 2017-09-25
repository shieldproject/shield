package disk

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshretry "github.com/cloudfoundry/bosh-utils/retrystrategy"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	"github.com/pivotal-golang/clock"
)

type partedPartitioner struct {
	logger      boshlog.Logger
	cmdRunner   boshsys.CmdRunner
	logTag      string
	timeService clock.Clock
}

func NewPartedPartitioner(logger boshlog.Logger, cmdRunner boshsys.CmdRunner, timeService clock.Clock) Partitioner {
	return partedPartitioner{
		logger:      logger,
		cmdRunner:   cmdRunner,
		logTag:      "PartedPartitioner",
		timeService: timeService,
	}
}

func (p partedPartitioner) Partition(devicePath string, partitions []Partition) error {
	existingPartitions, deviceFullSizeInBytes, err := p.getPartitions(devicePath)
	if err != nil {
		return bosherr.WrapErrorf(err, "Getting existing partitions of `%s'", devicePath)
	}

	if p.partitionsMatch(existingPartitions, partitions) {
		return nil
	}

	partitionStart := p.decideFirstPartitionStartingPoint(existingPartitions)

	if err = p.createEachPartition(partitions, partitionStart, deviceFullSizeInBytes, devicePath); err != nil {
		return err
	}

	if strings.Contains(devicePath, "/dev/mapper/") {
		if err = p.createMapperPartition(devicePath); err != nil {
			return err
		}
	}

	return nil
}

func (p partedPartitioner) GetDeviceSizeInBytes(devicePath string) (uint64, error) {
	p.logger.Debug(p.logTag, "Getting size of disk remaining after first partition")

	stdout, _, _, err := p.cmdRunner.RunCommand("parted", "-m", devicePath, "unit", "B", "print")
	if err != nil {
		return 0, bosherr.WrapErrorf(err, "Getting remaining size of `%s'", devicePath)
	}

	allLines := strings.Split(stdout, "\n")
	if len(allLines) < 3 {
		return 0, bosherr.Errorf("Getting remaining size of `%s'", devicePath)
	}

	partitionInfoLines := allLines[1:3]
	deviceInfo := strings.Split(partitionInfoLines[0], ":")
	deviceFullSizeInBytes, err := strconv.ParseUint(strings.TrimRight(deviceInfo[1], "B"), 10, 64)
	if err != nil {
		return 0, bosherr.WrapErrorf(err, "Getting remaining size of `%s'", devicePath)
	}

	firstPartitionInfo := strings.Split(partitionInfoLines[1], ":")
	firstPartitionEndInBytes, err := strconv.ParseUint(strings.TrimRight(firstPartitionInfo[2], "B"), 10, 64)
	if err != nil {
		return 0, bosherr.WrapErrorf(err, "Getting remaining size of `%s'", devicePath)
	}

	remainingSizeInBytes := deviceFullSizeInBytes - firstPartitionEndInBytes - 1

	return remainingSizeInBytes, nil
}

func (p partedPartitioner) partitionsMatch(existingPartitions []existingPartition, partitions []Partition) bool {
	if len(existingPartitions) != len(partitions) {
		return false
	}

	for index, partition := range partitions {
		existingPartition := existingPartitions[index]
		if !withinDelta(partition.SizeInBytes, existingPartition.SizeInBytes, p.convertFromMbToBytes(20)) {
			return false
		}
	}

	return true
}

func (p partedPartitioner) getPartitions(devicePath string) (partitions []existingPartition, deviceFullSizeInBytes uint64, err error) {
	stdout, _, _, err := p.runPartedPrint(devicePath)

	if err != nil {
		return partitions, deviceFullSizeInBytes, bosherr.WrapErrorf(err, "Running parted print")
	}

	allLines := strings.Split(stdout, "\n")
	if len(allLines) < 2 {
		return partitions, deviceFullSizeInBytes, bosherr.Errorf("Parsing existing partitions")
	}

	deviceInfo := strings.Split(allLines[1], ":")
	deviceFullSizeInBytes, err = strconv.ParseUint(strings.TrimRight(deviceInfo[1], "B"), 10, 64)
	if err != nil {
		return partitions, deviceFullSizeInBytes, bosherr.WrapErrorf(err, "Parsing device size")
	}

	partitionLines := allLines[2 : len(allLines)-1]

	for _, partitionLine := range partitionLines {
		// ignore PReP partition on ppc64le
		if strings.Contains(partitionLine, "prep") {
			continue
		}
		partitionInfo := strings.Split(partitionLine, ":")
		partitionIndex, err := strconv.Atoi(partitionInfo[0])

		if err != nil {
			return partitions, deviceFullSizeInBytes, bosherr.WrapErrorf(err, "Parsing existing partitions")
		}

		partitionStartInBytes, err := strconv.Atoi(strings.TrimRight(partitionInfo[1], "B"))
		if err != nil {
			return partitions, deviceFullSizeInBytes, bosherr.WrapErrorf(err, "Parsing existing partitions")
		}

		partitionEndInBytes, err := strconv.Atoi(strings.TrimRight(partitionInfo[2], "B"))
		if err != nil {
			return partitions, deviceFullSizeInBytes, bosherr.WrapErrorf(err, "Parsing existing partitions")
		}

		partitionSizeInBytes, err := strconv.Atoi(strings.TrimRight(partitionInfo[3], "B"))
		if err != nil {
			return partitions, deviceFullSizeInBytes, bosherr.WrapErrorf(err, "Parsing existing partitions")
		}

		partitions = append(
			partitions,
			existingPartition{
				Index:        partitionIndex,
				SizeInBytes:  uint64(partitionSizeInBytes),
				StartInBytes: uint64(partitionStartInBytes),
				EndInBytes:   uint64(partitionEndInBytes),
			},
		)
	}

	return partitions, deviceFullSizeInBytes, nil
}

func (p partedPartitioner) convertFromBytesToMb(sizeInBytes uint64) uint64 {
	return sizeInBytes / (1024 * 1024)
}

func (p partedPartitioner) convertFromMbToBytes(sizeInMb uint64) uint64 {
	return sizeInMb * 1024 * 1024
}

func (p partedPartitioner) convertFromKbToBytes(sizeInKb uint64) uint64 {
	return sizeInKb * 1024
}

func (p partedPartitioner) runPartedPrint(devicePath string) (stdout, stderr string, exitStatus int, err error) {
	stdout, stderr, exitStatus, err = p.cmdRunner.RunCommand("parted", "-m", devicePath, "unit", "B", "print")

	// If the error is not having a partition table, create one
	if err != nil && strings.Contains(err.Error(), "unrecognised disk label") {
		stdout, stderr, exitStatus, err = p.getPartitionTable(devicePath)

		if err != nil {
			return stdout, stderr, exitStatus, bosherr.WrapErrorf(err, "Parted making label")
		}

		return p.cmdRunner.RunCommand("parted", "-m", devicePath, "unit", "B", "print")
	}

	return stdout, stderr, exitStatus, err
}

func (p partedPartitioner) getPartitionTable(devicePath string) (stdout, stderr string, exitStatus int, err error) {
	return p.cmdRunner.RunCommand(
		"parted",
		"-s",
		devicePath,
		"mklabel",
		"gpt",
	)
}

func (p partedPartitioner) roundUp(numToRound, multiple uint64) uint64 {
	if multiple == 0 {
		return numToRound
	}
	remainder := numToRound % multiple
	if remainder == 0 {
		return numToRound
	}
	return numToRound + multiple - remainder
}

func (p partedPartitioner) roundDown(numToRound, multiple uint64) uint64 {
	if multiple == 0 {
		return numToRound
	}
	remainder := numToRound % multiple
	if remainder == 0 {
		return numToRound
	}
	return numToRound - remainder
}

func (p partedPartitioner) decideFirstPartitionStartingPoint(existingPartitions []existingPartition) uint64 {
	partitionStart := uint64(0)
	if len(existingPartitions) == 0 {
		partitionStart = uint64(513)
	} else {
		partitionStart = existingPartitions[len(existingPartitions)-1].EndInBytes + 1
	}

	alignmentInBytes := uint64(1048576)
	partitionStart = p.roundUp(partitionStart, alignmentInBytes)
	return partitionStart
}

func (p partedPartitioner) createEachPartition(partitions []Partition, partitionStart uint64, deviceFullSizeInBytes uint64, devicePath string) error {
	//For each Parition
	alignmentInBytes := uint64(1048576)
	for index, partition := range partitions {

		//Get end point for partition
		var partitionEnd uint64

		if partition.SizeInBytes == 0 {
			// If no partitions were specified, use the whole disk space
			partitionEnd = p.roundDown(deviceFullSizeInBytes-1, alignmentInBytes)
		} else {
			partitionEnd = partitionStart + partition.SizeInBytes
			// If the partition size is greater than the remaining space on disk, truncate the partition to whatever size is left
			if partitionEnd >= deviceFullSizeInBytes {
				partitionEnd = deviceFullSizeInBytes - 1
				p.logger.Info(p.logTag, "Partition %d would be larger than remaining space. Reducing size to %dB", index, partitionEnd-partitionStart)
			}
			partitionEnd = p.roundDown(partitionEnd, alignmentInBytes) - 1
		}

		// Create and run a retryable
		partitionRetryable := boshretry.NewRetryable(func() (bool, error) {
			_, _, _, err := p.cmdRunner.RunCommand(
				"parted",
				"-s",
				devicePath,
				"unit",
				"B",
				"mkpart",
				"primary",
				fmt.Sprintf("%d", partitionStart),
				fmt.Sprintf("%d", partitionEnd),
			)
			if err != nil {
				p.logger.Error(p.logTag, "Failed with an error: %s", err)
				//TODO: double check the output here. Does it make sense?
				return true, bosherr.WrapError(err, "Creating partition using parted")
			}
			p.logger.Info(p.logTag, "Successfully created partition %d on %s", index, devicePath)
			return false, nil
		})

		partitionRetryStrategy := NewPartitionStrategy(partitionRetryable, p.timeService, p.logger)
		err := partitionRetryStrategy.Try()

		if err != nil {
			return bosherr.WrapErrorf(err, "Partitioning disk `%s'", devicePath)
		}

		//increment
		partitionStart = p.roundUp(partitionEnd+1, alignmentInBytes)
	}
	return nil
}

func (p partedPartitioner) createMapperPartition(devicePath string) error {
	_, _, _, err := p.cmdRunner.RunCommand("/etc/init.d/open-iscsi", "restart")
	if err != nil {
		return bosherr.WrapError(err, "Shelling out to restart open-iscsi")
	}

	detectPartitionRetryable := boshretry.NewRetryable(func() (bool, error) {
		output, _, _, err := p.cmdRunner.RunCommand("dmsetup", "ls")
		if err != nil {
			return true, bosherr.WrapError(err, "Shelling out to dmsetup ls")
		}

		if strings.Contains(output, "No devices found") {
			return true, bosherr.Errorf("No devices found")
		}

		device := strings.TrimPrefix(devicePath, "/dev/mapper/")
		lines := strings.Split(strings.Trim(output, "\n"), "\n")
		for i := 0; i < len(lines); i++ {
			if match, _ := regexp.MatchString("-part1", lines[i]); match {
				if strings.Contains(lines[i], device) {
					p.logger.Info(p.logTag, "Succeeded in detecting partition %s", devicePath+"-part1")
					return false, nil
				}
			}
		}

		return true, bosherr.Errorf("Partition %s does not show up", devicePath+"-part1")
	})

	detectPartitionRetryStrategy := NewPartitionStrategy(detectPartitionRetryable, p.timeService, p.logger)
	return detectPartitionRetryStrategy.Try()
}
