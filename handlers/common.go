package handlers
import (
	"encoding/json"
	"log"
	"net/http"
)

type CustomUIResponse struct {
	Status  string      `json:"status,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func setupResponse(w *http.ResponseWriter) {
	(*w).Header().Set("Content-Type", "application/json")
	(*w).Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	(*w).Header().Set("Access-Control-Allow-Credentials", "true")
}

var SUCCESS_STRING = "Success"

func WriteError(w http.ResponseWriter, r *http.Request, code int, errresp interface{}) {
	log.Printf(
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
	log.Printf(
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

