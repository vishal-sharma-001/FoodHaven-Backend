package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	// "github.com/vishal-sharma-001/FoodHaven-backend.git/middleware"

	routes "github.com/vishal-sharma-001/FoodHaven-backend.git/routes"
)

func main(){
	port := ":8080"
	// fserver := http.FileServer(http.Dir("./dir"))

	// http.Handle("/", fserver)
	
	r := mux.NewRouter()

	routes.RegisterFoodRoutes(r)
	routes.RegisterRestaurantsRoutes(r)
	
	fmt.Printf("Starting the server at http://localhost%v \n", port)
	
	err := http.ListenAndServe(port, r)
	if(err != nil){
		fmt.Printf("%v",err)
	}
}