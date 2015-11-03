package api

import (
	"encoding/json"
	"net/http"
)

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
