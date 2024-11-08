package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

var SUCCESS_STRING = "Success"

func WriteError(w http.ResponseWriter, r *http.Request, code int, errresp interface{}) {
	fmt.Printf(
		"%s %s %v",
		r.Method,
		r.RequestURI,
		errresp,
	)
	w.WriteHeader(code)
	body, err := json.Marshal(errresp)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Write(body)
}

func WriteSuccessMessage(w http.ResponseWriter, r *http.Request, data interface{}) {
	fmt.Printf(
		"%s %s ",
		r.Method,
		r.RequestURI,
	)
	w.WriteHeader(http.StatusOK)
	body, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Write(body)
}
