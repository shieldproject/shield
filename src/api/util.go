package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
)

func match(req *http.Request, pattern string) bool {
	matched, _ := regexp.MatchString(
		fmt.Sprintf("^%s$", pattern),
		fmt.Sprintf("%s %s", req.Method, req.URL.Path))
	return matched
}

func JSON(w http.ResponseWriter, thing interface{}) {
	bytes, err := json.Marshal(thing)
	if err != nil {
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
