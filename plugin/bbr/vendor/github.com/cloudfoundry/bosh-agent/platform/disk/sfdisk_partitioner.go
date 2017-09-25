package disk

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"code.cloudfoundry.org/clock"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshretry "github.com/cloudfoundry/bosh-utils/retrystrategy"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type sfdiskPartitioner struct {
	logger      boshlog.Logger
	cmdRunner   boshsys.CmdRunner
	logTag      string
	timeService clock.Clock
}

func NewSfdiskPartitioner(logger boshlog.Logger, cmdRunner boshsys.CmdRunner, timeService clock.Clock) Partitioner {
	return sfdiskPartitioner{
		logger:      logger,
		cmdRunner:   cmdRunner,
		logTag:      "SfdiskPartitioner",
		timeService: timeService,
	}
}

func (p sfdiskPartitioner) Partition(devicePath string, partitions []Partition) error {
	partitionMatches, err := p.diskMatchesPartitions(devicePath, partitions)
	if err != nil {
		return err
	}
	if partitionMatches {
		p.logger.Info(p.logTag, "%s already partitioned as expected, skipping", devicePath)
		return nil
	}
	sfdiskPartitionTypes := map[PartitionType]string{
		PartitionTypeSwap:  "S",
		PartitionTypeLinux: "L",
	}

	sfdiskInput := ""
	for index, partition := range partitions {
		sfdiskPartitionType := sfdiskPartitionTypes[partition.Type]
		partitionSize := fmt.Sprintf("%d", p.convertFromBytesToMb(partition.SizeInBytes))

		if index == len(partitions)-1 {
			partitionSize = ""
		}

		sfdiskInput = sfdiskInput + fmt.Sprintf(",%s,%s\n", partitionSize, sfdiskPartitionType)
	}

	partitionRetryable := boshretry.NewRetryable(func() (bool, error) {
		_, _, _, err := p.cmdRunner.RunCommandWithInput(sfdiskInput, "sfdisk", "-uM", devicePath)
		if err != nil {
			p.logger.Error(p.logTag, "Failed with an error: %s", err)
			return true, bosherr.WrapError(err, "Shelling out to sfdisk")
		}
		p.logger.Info(p.logTag, "Succeeded in partitioning %s with %s", devicePath, sfdiskInput)
		return false, nil
	})

	partitionRetryStrategy := NewPartitionStrategy(partitionRetryable, p.timeService, p.logger)
	err = partitionRetryStrategy.Try()
	if err != nil {
		return err
	}
	if strings.Contains(devicePath, "/dev/mapper/") {
		_, _, _, err = p.cmdRunner.RunCommand("/etc/init.d/open-iscsi", "restart")
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
		err := detectPartitionRetryStrategy.Try()
		if err != nil {
			return err
		}
	}

	return nil
}

func (p sfdiskPartitioner) GetDeviceSizeInBytes(devicePath string) (uint64, error) {
	stdout, _, _, err := p.cmdRunner.RunCommand("sfdisk", "-s", devicePath)
	if err != nil {
		return 0, bosherr.WrapError(err, "Shelling out to sfdisk when getting device size")
	}

	sizeInKb, err := strconv.ParseUint(strings.Trim(stdout, "\n"), 10, 64)
	if err != nil {
		return 0, bosherr.WrapError(err, "Converting disk size to integer")
	}

	return p.convertFromKbToBytes(sizeInKb), nil
}

func (p sfdiskPartitioner) diskMatchesPartitions(devicePath string, partitionsToMatch []Partition) (bool, error) {
	existingPartitions, err := p.getPartitions(devicePath)
	if err != nil {
		return false, bosherr.WrapErrorf(err, "Getting partitions for %s", devicePath)
	}
	if len(existingPartitions) < len(partitionsToMatch) {
		return false, nil
	}

	remainingDiskSpace, err := p.GetDeviceSizeInBytes(devicePath)
	if err != nil {
		return false, bosherr.WrapErrorf(err, "Getting device size for %s", devicePath)
	}

	for index, partitionToMatch := range partitionsToMatch {
		if index == len(partitionsToMatch)-1 {
			partitionToMatch.SizeInBytes = remainingDiskSpace
		}

		existingPartition := existingPartitions[index]
		switch {
		case existingPartition.Type != partitionToMatch.Type:
			return false, nil
		case !withinDelta(existingPartition.SizeInBytes, partitionToMatch.SizeInBytes, p.convertFromMbToBytes(20)):
			return false, nil
		}

		remainingDiskSpace = remainingDiskSpace - partitionToMatch.SizeInBytes
	}

	return true, nil
}

func (p sfdiskPartitioner) getPartitions(devicePath string) ([]Partition, error) {
	stdout, _, _, err := p.cmdRunner.RunCommand("sfdisk", "-d", devicePath)
	if err != nil {
		return nil, bosherr.WrapError(err, "Shelling out to sfdisk when getting partitions")
	}

	partitions := []Partition{}

	allLines := strings.Split(stdout, "\n")
	if len(allLines) < 4 {
		return partitions, nil
	}

	partitionLines := allLines[3 : len(allLines)-1]

	for _, partitionLine := range partitionLines {
		partitionPath, partitionType := extractPartitionPathAndType(partitionLine)
		partition := Partition{Type: partitionType}

		if partition.Type != PartitionTypeEmpty {
			if strings.Contains(partitionPath, "/dev/mapper/") {
				partitionPath = partitionPath[0:len(partitionPath)-1] + "-part1"
			}
			size, err := p.GetDeviceSizeInBytes(partitionPath)
			if err == nil {
				partition.SizeInBytes = size
			}
		}

		partitions = append(partitions, partition)
	}
	return partitions, nil
}

var partitionTypesMap = map[string]PartitionType{
	"82": PartitionTypeSwap,
	"83": PartitionTypeLinux,
	"0":  PartitionTypeEmpty,
}

func extractPartitionPathAndType(line string) (partitionPath string, partitionType PartitionType) {
	partitionFields := strings.Fields(line)
	lastField := partitionFields[len(partitionFields)-1]

	sfdiskPartitionType := strings.Replace(lastField, "Id=", "", 1)

	partitionPath = partitionFields[0]
	partitionType = partitionTypesMap[sfdiskPartitionType]
	return
}

func (p sfdiskPartitioner) convertFromBytesToMb(sizeInBytes uint64) uint64 {
	return sizeInBytes / (1024 * 1024)
}

func (p sfdiskPartitioner) convertFromMbToBytes(sizeInMb uint64) uint64 {
	return sizeInMb * 1024 * 1024
}

func (p sfdiskPartitioner) convertFromKbToBytes(sizeInKb uint64) uint64 {
	return sizeInKb * 1024
}
