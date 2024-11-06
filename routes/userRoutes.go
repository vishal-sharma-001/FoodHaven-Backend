package routes

import (
	"net/http"

	"github.com/gorilla/mux"
	handlers "github.com/vishal-sharma-001/FoodHaven-backend.git/handlers"
)

func RegisterUserRoutes(r *mux.Router){
	r.NotFoundHandler = http.NotFoundHandler()
	
	r.HandleFunc("/users", handlers.GetUsers).Methods("GET")
}