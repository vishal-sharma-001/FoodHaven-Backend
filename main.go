package main

import (
	"fmt"
	"net/http"
	"github.com/gorilla/mux"
	// "github.com/vishal-sharma-001/FoodHaven-backend.git/database"
	// "github.com/vishal-sharma-001/FoodHaven-backend.git/middleware"
	// "go.mongodb.org/mongo-driver/mongo"
	
	routes "github.com/vishal-sharma-001/FoodHaven-backend.git/routes"
)

func main(){
	port := ":8080"
	// fserver := http.FileServer(http.Dir("./dir"))

	// http.Handle("/", fserver)

	r := mux.NewRouter()

	routes.RegisterFoodRoutes(r)
	
	fmt.Printf("Starting the server at port http://localhost%v \n", port)
	
	err := http.ListenAndServe(port, r)
	if(err != nil){
		fmt.Errorf("%v",err)
	}
}