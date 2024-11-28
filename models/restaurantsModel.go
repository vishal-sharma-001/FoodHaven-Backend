package models


type Restaurants struct {
    Id           int     `json:"id"`
    Name         string  `json:"name" validate:"required,min=1,max=100"`
    Rating       float64 `json:"rating" validate:"min=0,max=5"`
    Cuisine      string  `json:"cuisine"`
    DeliveryTime int     `json:"deliverytime"`
    Offers       string  `json:"offers"`
    Locality     string  `json:"locality"`
    CloudImageID string  `json:"cloudimageid"`
    CostForTwo   float64 `json:"costfortwo"` // changed to float64
    Veg          bool    `json:"veg"`
}
