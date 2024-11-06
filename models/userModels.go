package models

type Users struct {
	UserId    int    `json:"uid" validate:"required,min=1,max=100"`
	FirstName string `json:"firstname" validate:"required,min=1,max=100"`
	LastName  string `json:"lastname" validate:"min=1,max=100"`
	Email     string `json:"email" validate:"required,min=6,max=100"`
	Password  string `json:"password" validate:"required,min=6,max=100"`
	Phone     string `json:"phone" validate:"required,min=6,max=100"`
	Avatar    string `json:"avatar"`
}
