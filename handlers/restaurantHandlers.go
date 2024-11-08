package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	db "github.com/vishal-sharma-001/FoodHaven-Backend/database"
	models "github.com/vishal-sharma-001/FoodHaven-Backend/models"
)

var fetchRestaurantsList = "SELECT id, name, rating, cuisine, deliverytime, offers, locality, cloudimageid, costfortwo, veg FROM restaurantsdata WHERE city = $1"
var fetchCitiesQuery = "select distinct city from restaurantsdata"

func GetRestaurants(w http.ResponseWriter, r *http.Request) {
	city := r.URL.Query().Get("city")

	if city == "" {
        http.Error(w, "city parameter is required", http.StatusBadRequest)
        return
    }

	setupResponse(&w, r)

	var (
		err         error
		response    CustomUIResponse
		restaurants []models.Restaurants
	)

	dbClient, err := db.ConnectDB()
	if err != nil {
		fmt.Printf("Could not connect to the database: %v", err)
	}
	defer dbClient.Close()

	restaurants, err = fetchRestaurants(dbClient, city)
	if err != nil {
		response.Message = fmt.Sprintf("Error fetching restaurants list. Error: [%s]", err.Error())
		fmt.Printf(response.Message)
		WriteError(w, r, http.StatusInternalServerError, response)
	}

	response.Status = SUCCESS_STRING
	response.Message = SUCCESS_STRING
	response.Data = restaurants
	WriteSuccessMessage(w, r, response)
}

func GetCities(w http.ResponseWriter, r *http.Request){
	setupResponse(&w, r)

	var (
		err         error
		response    CustomUIResponse
		cities []string
	)

	dbClient, err := db.ConnectDB()
	if err != nil {
		fmt.Printf("Could not connect to the database: %v", err)
	}
	defer dbClient.Close()

	cities, err = fetchCities(dbClient)
	if err != nil {
		response.Message = fmt.Sprintf("Error fetching restaurants list. Error: [%s]", err.Error())
		fmt.Printf(response.Message)
		WriteError(w, r, http.StatusInternalServerError, response)
	}

	response.Status = SUCCESS_STRING
	response.Message = SUCCESS_STRING
	response.Data = cities
	WriteSuccessMessage(w, r, response)
}

func fetchCities(dbClient *sql.DB)(cities [] string, err error){

	rows, err := dbClient.Query(fetchCitiesQuery)
	if err != nil {
		fmt.Printf("Error executing sql command [%v]", err.Error())
		return cities, err
	}

	defer rows.Close()

	for rows.Next() {
		var city string

		if err := rows.Scan(&city); err != nil {
			fmt.Printf("Error scanning result rows: [%v]\n", err)
			return cities, err
		}

		cities = append(cities, city)
	}
	return cities, err
}

func fetchRestaurants(dbClient *sql.DB, city string) (restaurants []models.Restaurants, err error) {
	var args []interface{}
	args = append(args, city)

	rows, err := dbClient.Query(fetchRestaurantsList, args...)
	if err != nil {
		fmt.Printf("Error executing sql command [%v]", err.Error())
		return restaurants, err
	}
	defer rows.Close()

	for rows.Next() {
		var r models.Restaurants

		if err := rows.Scan(&r.Id, &r.Name, &r.Rating, &r.Cuisine, &r.DeliveryTime, &r.Offers, &r.Locality, &r.CloudImageID, &r.CostForTwo, &r.Veg); err != nil {
			fmt.Printf("Error scanning result rows: [%v]\n", err)
			return restaurants, err
		}

		restaurants = append(restaurants, r)
	}
	return restaurants, err
}
