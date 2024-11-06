package models


type Restaurants struct{
	Id  int `json:"id"`
	Name string `json:"name" validate:"required,min=1,max=100"`
	Ratings string `json:"ratings" validate:"min=0,max=5"`
	Type string `json:"type"`
}
