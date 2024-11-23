package models


type Restaurants struct {
	Id           int    `json:"id"`
	Name         string `json:"name" validate:"required,min=1,max=100"`
	Rating       int    `json:"rating" validate:"min=0,max=5"`
	Cuisine      string `json:"cuisine"`
	DeliveryTime int    `json:"deliverytime"`
	Offers       string `json:"offers"`
	Locality     string `json:"locality"`
	CloudImageID string `json:"cloudimageid"`
	CostForTwo   int    `json:"costfortwo"`
	Veg          bool   `json:"veg"`
}
