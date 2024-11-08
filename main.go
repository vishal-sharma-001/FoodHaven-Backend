package main

import (
	"fmt"
	"net/http"

	"github.com/go-acme/lego/log"
	"github.com/gorilla/mux"

	// "github.com/vishal-sharma-001/FoodHaven-Backend/middleware"

	routes "github.com/vishal-sharma-001/FoodHaven-Backend/routes"
)

func main() {

	port := ":8080"
	router := mux.NewRouter().StrictSlash(true)

	routes.RegisterFoodRoutes(router)
	routes.RegisterRestaurantsRoutes(router)

	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./FoodHavenUI")))

	fmt.Printf("Starting the server at http://localhost%v \n", port)

	err := http.ListenAndServe(port, router)

	if err != nil {
		log.Printf("Error [%s] in starting server", err.Error())
	}
}
