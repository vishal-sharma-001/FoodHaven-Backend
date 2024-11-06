package handlers

import (
	"fmt"
	"net/http"
)



func GetFoodList(w http.ResponseWriter, r *http.Request){
	fmt.Println("Food List......")
}