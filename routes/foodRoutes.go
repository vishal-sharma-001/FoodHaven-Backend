package routes

import (
	"net/http"

	"github.com/gorilla/mux"
	handlers "github.com/vishal-sharma-001/FoodHaven-Backend/handlers"
)

func RegisterFoodRoutes(r *mux.Router) {
	r.NotFoundHandler = http.NotFoundHandler()

	r.HandleFunc("/fooditems", handlers.GetFoodList).Methods("GET")
}
