package main

import (
	"fmt"
	"net/http"
)

func main(){
	port := ":8080"
	// fserver := http.FileServer(http.Dir("./dir"))

	// http.Handle("/", fserver)

	mux := http.NewServeMux()
	
	fmt.Printf("Starting the server at port http://localhost%v \n", port)
	err := http.ListenAndServe(port, mux)
	if(err != nil){
		fmt.Errorf("%v",err)
	}
}