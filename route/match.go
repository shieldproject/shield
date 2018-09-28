package route

import (
	"fmt"
	"net/http"
	"regexp"
)

type matcher func(req *http.Request) ([]string, bool)

var anySlug *regexp.Regexp

func init() {
	anySlug = regexp.MustCompile(":[a-z]+")
}

func newMatch(pat string) matcher {
	pat = anySlug.ReplaceAllString(pat, "([^/]+)")

	re := regexp.MustCompile(fmt.Sprintf("^%s$", pat))
	return func(req *http.Request) ([]string, bool) {
		m := re.FindStringSubmatch(fmt.Sprintf("%s %s", req.Method, req.URL.Path))
		return m, m != nil
	}
}
