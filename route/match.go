package route

import (
	"fmt"
	"net/http"
	"regexp"
)

type matcher func(req *http.Request) ([]string, bool)

var (
	uuidSlug *regexp.Regexp
	anySlug  *regexp.Regexp
)

func init() {
	uuidSlug = regexp.MustCompile(":uuid")
	anySlug = regexp.MustCompile(":[a-z]+")
}

func newMatch(pat string) matcher {
	pat = uuidSlug.ReplaceAllString(pat, "([a-fA-F0-9-]+)")
	pat = anySlug.ReplaceAllString(pat, "([^/]+)")

	re := regexp.MustCompile(fmt.Sprintf("^%s$", pat))
	return func(req *http.Request) ([]string, bool) {
		m := re.FindStringSubmatch(fmt.Sprintf("%s %s", req.Method, req.URL.Path))
		return m, m != nil
	}
}
