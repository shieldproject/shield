package api

import (
	"fmt"
	"os"
)

type YesNo struct {
	On  bool
	Yes bool
}

func MaybeString(tf string) YesNo {
	if tf == "" {
		return YesNo{}
	}
	return Maybe(tf == "t")
}

func Maybe(tf bool) YesNo {
	return YesNo{On: true, Yes: tf}
}

func MaybeBools(yes bool, no bool) YesNo {
	if yes {
		return Maybe(true)
	}
	if no {
		return Maybe(false)
	}
	return YesNo{} // unspecified
}

func No() YesNo {
	return Maybe(false)
}

func Yes() YesNo {
	return Maybe(true)
}

func Opposite(yn YesNo) YesNo {
	yn.Yes = !yn.Yes
	return yn
}

func (yn *YesNo) Given() bool {
	if yn == nil {
		return false
	}
	return yn.On
}

func ShieldURI(p string, args ...interface{}) *URL {
	endpoint, err := Cfg.SecureBackendURI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	path := fmt.Sprintf(p, args...)
	u, err := ParseURL(fmt.Sprintf("%s%s", endpoint, path))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	return u
}
