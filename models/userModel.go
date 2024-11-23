package models

type User struct {
	Id       int    `json:"id" validate:"required"`
	Name     string `json:"name" validate:"required, min=5 max=100"`
	Email    string `json:"email" validate:"required, min=5 max=100"`
	Phone    string `json:"phone" validate:"required, min=5 max=100"`
	Password string `json:"password" validate:"required, min=5 max=100"`
}

type Address struct {
    ID          int    `json:"id"`
    UserID      int    `json:"user_id"`
    Name        string `json:"name"`
    Street      string `json:"street"`
    City        string `json:"city"`
    PostalCode  string `json:"postalCode"`
    Phone       string `json:"phone"`
    IsPrimary   bool   `json:"is_primary"`
}
