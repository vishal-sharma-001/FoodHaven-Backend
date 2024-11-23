package models

type FoodItems struct {
	Id           int     `json:"id"`
	Name         string  `json:"name"`
	Price        float64 `json:"price"`
	Description  string  `json:"description"`
	CloudImageID string  `json:"cloudimageid"`
	Category     string  `json:"category"`
}
