package s3

import (
	"fmt"
)

type Bytes int64

func (b Bytes) String() string {
	if b < 1<<10 {
		return b.Bytes()
	}
	if b < 1<<20 {
		return b.Kilobytes()
	}
	if b < 1<<30 {
		return b.Megabytes()
	}
	if b < 1<<40 {
		return b.Gigabytes()
	}
	if b < 1<<50 {
		return b.Terabytes()
	}
	if b < 1<<60 {
		return b.Petabytes()
	}
	return b.Exabytes()
}

func (b Bytes) Bytes() string {
	return fmt.Sprintf("%db", b)
}

func (b Bytes) Kilobytes() string {
	return fmt.Sprintf("%dk", b/(1<<10))
}

func (b Bytes) Megabytes() string {
	return fmt.Sprintf("%dm", b/(1<<20))
}

func (b Bytes) Gigabytes() string {
	return fmt.Sprintf("%dg", b/(1<<30))
}

func (b Bytes) Terabytes() string {
	return fmt.Sprintf("%dt", b/(1<<40))
}

func (b Bytes) Petabytes() string {
	return fmt.Sprintf("%dp", b/(1<<50))
}

func (b Bytes) Exabytes() string {
	return fmt.Sprintf("%dx", b/(1<<60))
}
