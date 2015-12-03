package api_agent

import (
	"fmt"
	"github.com/spf13/viper"
)

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
