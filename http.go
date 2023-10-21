package main

import (
	"encoding/json"
	"net/http"
)

func RespondJson(w http.ResponseWriter, r *http.Request, res HTTPResponse) {
	jsonResponse, _ := json.Marshal(res)
	w.WriteHeader(res.Status)
	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write(jsonResponse)
	if err != nil {
		LogErr(r, err)
	}
}
