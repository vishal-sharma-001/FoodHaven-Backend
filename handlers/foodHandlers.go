package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	db "github.com/vishal-sharma-001/FoodHaven-Backend/database"
	"github.com/vishal-sharma-001/FoodHaven-Backend/models"
)

var fetchfoodItemsQuery = "SELECT id, name, price, description, cloudimageid, category FROM FoodItems WHERE cloudimageid = $1"

func GetFoodList(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("cloudimageid")

	if id == "" {
		http.Error(w, "cloudimageid parameter is required", http.StatusBadRequest)
		return
	}

	setupResponse(&w)

	var response  CustomUIResponse

	dbClient, err := db.ConnectDB()
	if err != nil {
		log.Printf("Could not connect to the database: %v", err)
	}
	defer dbClient.Close()

	foodItems, err := fetchFoodItems(dbClient, id)
	if err != nil {
		response.Message = fmt.Sprintf("Error fetching food list. Error: [%s]", err.Error())
		log.Printf(response.Message)
		WriteError(w, r, http.StatusInternalServerError, response)
		return
	}

	response.Status = SUCCESS_STRING
	response.Message = SUCCESS_STRING
	response.Data = foodItems
	WriteSuccessMessage(w, r, response)
}

func fetchFoodItems(dbClient *sql.DB, id string) (map[string][]models.FoodItems, error) {
	var args []interface{}
	args = append(args, id)

	rows, err := dbClient.Query(fetchfoodItemsQuery, args...)
	if err != nil {
		log.Printf("Error executing SQL command: [%v]", err.Error())
		return nil, err
	}
	defer rows.Close()

	categorizedFoodItems := make(map[string][]models.FoodItems)

	for rows.Next() {
		var item models.FoodItems
		if err := rows.Scan(&item.Id, &item.Name, &item.Price, &item.Description, &item.CloudImageID, &item.Category); err != nil {
			log.Printf("Error scanning result rows: [%v]\n", err)
			return nil, err
		}
		categorizedFoodItems[item.Category] = append(categorizedFoodItems[item.Category], item)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error with rows iteration: [%v]", err.Error())
		return nil, err
	}

	return categorizedFoodItems, nil
}
