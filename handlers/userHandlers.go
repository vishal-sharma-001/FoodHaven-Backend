package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	models "github.com/vishal-sharma-001/FoodHaven-backend.git/models"
)

type CustomUIResponse struct {
	Status  string      `json:"status,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func setupResponse(w *http.ResponseWriter, req *http.Request) {
	(*w).Header().Set("content-type", "application/json")
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}


func GetUsers(w http.ResponseWriter, r *http.Request){
	setupResponse(&w, r)

	var (
		err          error
		response     CustomUIResponse
		usersList []models.Users
	)


	usersList, err = fetchUsersList(dbClient)
	if err != nil {
		response.Message = fmt.Sprintf("Error fetching warnings list. Error: [%s]", err.Error())
		log.Print(response.Message)
		WriteError(w, r, http.StatusInternalServerError, response)
	}

	response.Status = SUCCESS_STRING
	response.Message = SUCCESS_STRING
	response.Data = usersList
	WriteSuccessMessage(w, r, response)
}

func fetchUsersList( dbclient *sql.DB) ([]models.Users, error){
	var usersList []models.Users
	
	return usersList, nil
}