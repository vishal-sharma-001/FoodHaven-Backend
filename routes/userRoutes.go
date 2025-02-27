package routes

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	handlers "github.com/vishal-sharma-001/FoodHaven-Backend/handlers"
)

func RegisterUserRoutes(r *mux.Router, store *sessions.CookieStore) {
	r.NotFoundHandler = http.NotFoundHandler()

	r.HandleFunc("/user/signup", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleSignUp(w, r, store)
	}).Methods("POST", "OPTIONS")

	r.HandleFunc("/user/login", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleLogIn(w, r, store)
	}).Methods("POST", "OPTIONS")

}

func RegisterProtectedUserRoutes(r *mux.Router, store *sessions.CookieStore) {
	r.NotFoundHandler = http.NotFoundHandler()

	r.HandleFunc("/user/getuser", handlers.HandleGetUser).Methods("GET", "OPTIONS")

	r.HandleFunc("/user/edit", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleEditUser(w, r, store)
	}).Methods("POST", "OPTIONS")

	r.HandleFunc("/user/logout", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleLogOut(w, r, store)
	}).Methods("POST", "OPTIONS")

	r.HandleFunc("/user/getcart", handlers.FetchCart).Methods("GET", "OPTIONS")

	r.HandleFunc("/user/getaddresses", handlers.GetUserAddresses).Methods("GET", "OPTIONS")
	r.HandleFunc("/user/addaddress", handlers.HandleAddAddress).Methods("POST", "OPTIONS")
	r.HandleFunc("/user/editaddress/{id}", handlers.HandleEditAddress).Methods("PUT", "OPTIONS")
	r.HandleFunc("/user/deleteaddress/{id}", handlers.HandleDeleteAddress).Methods("DELETE", "OPTIONS")

	r.HandleFunc("/user/synccart/{cart_id}", handlers.SyncCart).Methods("POST", "OPTIONS")

	r.HandleFunc("/payment/create-checkout-session", handlers.CreateCheckoutSession).Methods("POST", "OPTIONS")
	r.HandleFunc("/payment/session-status", handlers.RetrieveCheckoutSession).Methods("GET", "OPTIONS")

	r.HandleFunc("/user/fetchorders", handlers.FetchOrders).Methods("GET", "OPTIONS")
}

