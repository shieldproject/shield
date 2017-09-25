package fmt

import (
	"fmt"
	"math"
	"time"
)

func Duration(duration time.Duration) string {
	totalSeconds := math.Floor(duration.Seconds())
	hours := math.Floor(totalSeconds / 3600)
	minutes := math.Floor((totalSeconds - hours*3600) / 60)
	seconds := math.Floor(totalSeconds - (hours * 3600) - minutes*60)

	return fmt.Sprintf("%02.f:%02.f:%02.f", hours, minutes, seconds)
}
