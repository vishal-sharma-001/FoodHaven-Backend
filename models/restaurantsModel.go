package models


type Restaurants struct {
	Id           int    `json:"id"`
	Name         string `json:"name" validate:"required,min=1,max=100"`
	Rating       int    `json:"rating" validate:"min=0,max=5"`
	Cuisine      string `json:"cuisine"`
	DeliveryTime int    `json:"deliverytime"`  // Delivery time is now an integer (in minutes)
	Offers       string `json:"offers"`
	Locality     string `json:"locality"`
	CloudImageID string `json:"cloudimageid"`
	CostForTwo   int    `json:"costfortwo"`   // Added CostForTwo as an integer (in currency)
	Veg          bool   `json:"veg"`          // Added Veg as a boolean (true or false)
}
