package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	models "github.com/vishal-sharma-001/FoodHaven-backend.git/models"
)

var fetchRestaurantsList = "select * from restaurantsData";


func GetRestaurants(w http.ResponseWriter, r *http.Request){
	setupResponse(&w, r)

	var (
		err          error
		response     CustomUIResponse
		restaurants []models.Restaurants
	)

	restaurants, err = fetchRestaurants(dbClient)
	if err != nil {
		response.Message = fmt.Sprintf("Error fetching restaurants list. Error: [%s]", err.Error())
		fmt.Errorf(response.Message)
		WriteError(w, r, http.StatusInternalServerError, response)
	}

	response.Status = SUCCESS_STRING
	response.Message = SUCCESS_STRING
	response.Data = restaurants
	WriteSuccessMessage(w, r, response)
}

func fetchRestaurants(dbClient *sql.DB) (restaurants []models.Restaurants, err error) {

	rows, err := dbClient.Query(fetchRestaurantsList)
	if err != nil {
		fmt.Errorf("Error executing sql command [%v]", err.Error())
		return restaurants, err
	}
	defer rows.Close()

	for rows.Next() {
		var r models.Restaurants
		if err := rows.Scan(&r.Id, &r.Name, r.Ratings, r.Type); err != nil {
			fmt.Errorf("Error scanning result rows: [%v]", err)
			return restaurants, err
		}
		restaurants = append(restaurants, r)
	}
	return restaurants, err
}