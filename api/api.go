package api

import (
	"fmt"
	"github.com/spf13/viper"
	"time"
)

type Timestamp time.Time

func (t Timestamp) MarshalJSON() ([]byte, error) {
	if time.Time(t).IsZero() {
		return []byte("\"\""), nil
	}
	stamp := fmt.Sprintf("\"%s\"", time.Time(t).Format("2006-01-02 15:04:05"))
	return []byte(stamp), nil
}

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

func No() YesNo {
	return Maybe(false)
}

func Yes() YesNo {
	return Maybe(true)
}

func (yn *YesNo) Given() bool {
	if yn == nil {
		return false
	}
	return yn.On
}

func ShieldURI(p string, args ...interface{}) *URL {
	path := fmt.Sprintf(p, args...)
	scheme := "http"
	if viper.GetBool("ShieldSSL") {
		scheme = "https"
	}

	u, err := ParseURL(fmt.Sprintf("%s://%s:%s%s",
		scheme,
		viper.GetString("ShieldServer"),
		viper.GetString("ShieldPort"),
		path))
	if err != nil {
		panic(err)
	}
	return u
}
