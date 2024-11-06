package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	db "github.com/vishal-sharma-001/FoodHaven-backend.git/database"
)

var dbClient *sql.DB
var SUCCESS_STRING = "Success"
func init(){
	var err error
	dbClient, err = db.ConnectDB()
	if err != nil {
		fmt.Printf("Could not connect to the database: %v", err)
	}
	defer dbClient.Close()
}

func WriteError(w http.ResponseWriter, r *http.Request, code int, errresp interface{}) {
	fmt.Errorf(
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