// Jamie: This contains the go source code that will become shield.

package supervisor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

func match(req *http.Request, pattern string) bool {
	matched, _ := regexp.MatchString(
		fmt.Sprintf("^%s$", pattern),
		fmt.Sprintf("%s %s", req.Method, req.URL.Path))
	return matched
}

func bail(w http.ResponseWriter, e error) {
	w.WriteHeader(500)
	fmt.Printf("ERROR: <%s>\n", e)
	return
}

func JSON(w http.ResponseWriter, thing interface{}) {
	bytes, err := json.Marshal(thing)
	if err != nil {
		fmt.Printf("ERROR: <%s>\n", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(bytes)
	return
}

func JSONLiteral(w http.ResponseWriter, thing string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write([]byte(thing))
	return
}

func paramEquals(req *http.Request, name string, value string) bool {
	actual, set := req.URL.Query()[name]
	return set && actual[0] == value
}

func paramValue(req *http.Request, name string, defval string) string {
	value, set := req.URL.Query()[name]
	if set {
		return value[0]
	}
	return defval
}

func paramDate(req *http.Request, name string) *time.Time {
	//utc, _ := time.LoadLocation("UTC")

	value, set := req.URL.Query()[name]
	if !set {
		return nil
	}

	t, err := time.Parse("20060102", value[0])
	if err != nil {
		return nil
	}
	return &t
}
