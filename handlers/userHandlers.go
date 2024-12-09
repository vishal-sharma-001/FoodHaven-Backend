package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"

	db "github.com/vishal-sharma-001/FoodHaven-Backend/database"
	"github.com/vishal-sharma-001/FoodHaven-Backend/middleware"
	"github.com/vishal-sharma-001/FoodHaven-Backend/models"
)

const stripeSecretKey = "sk_test_51QTm8eLhrle3XiFesp5JSKKB0oMcGiRjpYSPlrt9FJ9RZjn3WpvW71HypVJfdhYNPOw5KjFy13JFK4q4ICPy4LqB00YHCpQVT6"
const domain = "https://foodhaven.run.place"

func init() {
	stripe.Key = stripeSecretKey
}

func HandleSignUp(w http.ResponseWriter, r *http.Request, store *sessions.CookieStore) {
	setupResponse(&w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		WriteError(w, r, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if user.Name == "" || user.Email == "" || user.Phone == "" || user.Password == "" {
		WriteError(w, r, http.StatusBadRequest, "All fields are required")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	dbClient, err := db.ConnectDB()
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Database connection error")
		return
	}
	defer dbClient.Close()

	var userID int
	query := "INSERT INTO users (name, email, phone, password) VALUES ($1, $2, $3, $4) RETURNING id"
	err = dbClient.QueryRow(query, user.Name, user.Email, user.Phone, hashedPassword).Scan(&userID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			if strings.Contains(pqErr.Message, "users_email_key") {
				WriteError(w, r, http.StatusConflict, "Email is already registered")
			} else if strings.Contains(pqErr.Message, "users_phone_key") {
				WriteError(w, r, http.StatusConflict, "Phone number is already registered")
			} else {
				WriteError(w, r, http.StatusInternalServerError, "Database error")
			}
			return
		}
		WriteError(w, r, http.StatusInternalServerError, "Failed to register user")
		return
	}

	session, err := store.Get(r, "user_session")
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Failed to create session")
		return
	}

	session.Values["userId"] = userID
	session.Values["email"] = user.Email
	session.Values["name"] = user.Name
	session.Values["phone"] = user.Phone

	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   24 * 60 * 60,
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	if err := session.Save(r, w); err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Failed to save session")
		return
	}

	user.Id = userID
	WriteSuccessMessage(w, r, user)
}

func HandleLogIn(w http.ResponseWriter, r *http.Request, store *sessions.CookieStore) {
	setupResponse(&w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		WriteError(w, r, http.StatusBadRequest, "Invalid request payload")
		return
	}

	dbClient, err := db.ConnectDB()
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Database connection error")
		return
	}
	defer dbClient.Close()

	var user models.User
	err = dbClient.QueryRow("SELECT id, name, email, phone, password FROM users WHERE email = $1", credentials.Email).Scan(&user.Id, &user.Name, &user.Email, &user.Phone, &user.Password)
	if err != nil {
		WriteError(w, r, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password)); err != nil {
		WriteError(w, r, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	session, err := store.Get(r, "user_session")
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Failed to create session")
		return
	}

	session.Values["userId"] = user.Id
	session.Values["email"] = user.Email
	session.Values["name"] = user.Name
	session.Values["phone"] = user.Phone

	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400,                // 1 day
		HttpOnly: true,                 // This ensures the cookie is only accessible via HTTP
		Secure:   false,                // Set to true if using HTTPS
		SameSite: http.SameSiteLaxMode, // Allows cross-origin cookies
	}

	log.Printf("------->Session Values: %v", session.Values)

	if err := session.Save(r, w); err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Failed to save session")
		return
	}

	WriteSuccessMessage(w, r, user)
}

func HandleGetUser(w http.ResponseWriter, r *http.Request) {
	setupResponse(&w)

	user, ok := r.Context().Value(middleware.ContextKeyUser).(models.User)
	if !ok {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}
	WriteSuccessMessage(w, r, user)
}

func HandleLogOut(w http.ResponseWriter, r *http.Request, store *sessions.CookieStore) {
	setupResponse(&w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	session, err := store.Get(r, "user_session")
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Failed to retrieve session")
		return
	}

	session.Values = map[interface{}]interface{}{}
	session.Options.MaxAge = -1

	if err := session.Save(r, w); err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Failed to clear session")
		return
	}

	WriteSuccessMessage(w, r, "Logged out successfully")
}

func HandleEditUser(w http.ResponseWriter, r *http.Request, store *sessions.CookieStore) {
	setupResponse(&w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		WriteError(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var updatedUser models.User
	if err := json.NewDecoder(r.Body).Decode(&updatedUser); err != nil {
		WriteError(w, r, http.StatusBadRequest, "Invalid request payload")
		return
	}

	user, ok := r.Context().Value(middleware.ContextKeyUser).(models.User)
	if !ok || user.Id == 0 {
		WriteError(w, r, http.StatusUnauthorized, "User not found in context")
		return
	}

	dbClient, err := db.ConnectDB()
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Database connection error")
		return
	}
	defer dbClient.Close()

	query := `
		UPDATE users 
		SET name = $1, email = $2, phone = $3 
		WHERE id = $4 
		RETURNING id
	`
	err = dbClient.QueryRow(query, updatedUser.Name, updatedUser.Email, updatedUser.Phone, user.Id).
		Scan(&updatedUser.Id)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Failed to update user information")
		return
	}

	session, err := store.Get(r, "user_session")
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Failed to retrieve session")
		return
	}

	session.Values["name"] = updatedUser.Name
	session.Values["email"] = updatedUser.Email
	session.Values["phone"] = updatedUser.Phone

	if err := session.Save(r, w); err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Failed to update session")
		return
	}

	WriteSuccessMessage(w, r, updatedUser)
}

func GetUserAddresses(w http.ResponseWriter, r *http.Request) {
	setupResponse(&w)

	user, ok := r.Context().Value(middleware.ContextKeyUser).(models.User)
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "User not found in context")
		return
	}

	dbClient, err := db.ConnectDB()
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Database connection error")
		return
	}
	defer dbClient.Close()

	rows, err := dbClient.Query(`SELECT id, user_id, name, street, city, postal_code, phone, is_primary FROM addresses WHERE user_id = $1`, user.Id)

	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Database query error")
		return
	}
	defer rows.Close()

	var addresses []models.Address

	for rows.Next() {
		var address models.Address
		err := rows.Scan(&address.ID, &address.UserID, &address.Name, &address.Street, &address.City, &address.PostalCode, &address.Phone, &address.IsPrimary)
		if err != nil {
			WriteError(w, r, http.StatusInternalServerError, "Error scanning row")
			return
		}
		addresses = append(addresses, address)
	}

	if err := rows.Err(); err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Error reading rows")
		return
	}

	WriteSuccessMessage(w, r, addresses)
}

func HandleAddAddress(w http.ResponseWriter, r *http.Request) {
	setupResponse(&w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		WriteError(w, r, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	user, ok := r.Context().Value(middleware.ContextKeyUser).(models.User)
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "User not found in context")
		return
	}

	var address models.Address
	if err := json.NewDecoder(r.Body).Decode(&address); err != nil {
		WriteError(w, r, http.StatusBadRequest, "Invalid request payload")
		return
	}

	dbClient, err := db.ConnectDB()
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Database connection error")
		return
	}
	defer dbClient.Close()

	err = dbClient.QueryRow(`
		INSERT INTO addresses (user_id, name, street, city, postal_code, phone, is_primary)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		user.Id, address.Name, address.Street, address.City, address.PostalCode, address.Phone, address.IsPrimary).Scan(&address.ID)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Failed to save address")
		return
	}

	address.UserID = user.Id
	WriteSuccessMessage(w, r, address)
}

func HandleEditAddress(w http.ResponseWriter, r *http.Request) {
	setupResponse(&w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPut {
		WriteError(w, r, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	user, ok := r.Context().Value(middleware.ContextKeyUser).(models.User)
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "User not found in context")
		return
	}

	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		WriteError(w, r, http.StatusBadRequest, "Address ID is required in the URL")
		return
	}

	addressID, err := strconv.Atoi(idStr)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, "Invalid address ID")
		return
	}

	var address models.Address
	if err := json.NewDecoder(r.Body).Decode(&address); err != nil {
		WriteError(w, r, http.StatusBadRequest, "Invalid request payload")
		return
	}

	address.ID = addressID

	dbClient, err := db.ConnectDB()
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Database connection error")
		return
	}
	defer dbClient.Close()

	query := `
		UPDATE addresses 
		SET name = $1, street = $2, city = $3, postal_code = $4, phone = $5, is_primary = $6 
		WHERE id = $7 AND user_id = $8`
	result, err := dbClient.Exec(query, address.Name, address.Street, address.City, address.PostalCode, address.Phone, address.IsPrimary, address.ID, user.Id)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Failed to update address")
		return
	}

	affectedRows, err := result.RowsAffected()
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Failed to retrieve rows affected")
		return
	}

	if affectedRows == 0 {
		WriteError(w, r, http.StatusNotFound, "Address not found or not authorized")
		return
	}

	WriteSuccessMessage(w, r, address)
}

func HandleDeleteAddress(w http.ResponseWriter, r *http.Request) {
	setupResponse(&w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodDelete {
		WriteError(w, r, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	user, ok := r.Context().Value(middleware.ContextKeyUser).(models.User)
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "User not found in context")
		return
	}

	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		WriteError(w, r, http.StatusBadRequest, "Address ID is required in the URL")
		return
	}

	addressID, err := strconv.Atoi(idStr)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, "Invalid address ID")
		return
	}

	dbClient, err := db.ConnectDB()
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Database connection error")
		return
	}
	defer dbClient.Close()

	query := `DELETE FROM addresses WHERE id = $1 AND user_id = $2`
	result, err := dbClient.Exec(query, addressID, user.Id)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Failed to delete address")
		return
	}

	affectedRows, err := result.RowsAffected()
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Failed to retrieve rows affected")
		return
	}

	if affectedRows == 0 {
		WriteError(w, r, http.StatusNotFound, "Address not found or not authorized")
		return
	}

	WriteSuccessMessage(w, r, map[string]string{"message": "Address deleted successfully"})
}

func CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	log.Println("--------------------Handling Initiate Request--------------------")
	setupResponse(&w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		WriteError(w, r, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	var req models.PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, "Invalid request body")
		return
	}

	lineItems := []*stripe.CheckoutSessionLineItemParams{}
	for _, item := range req.Items {
		lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency: stripe.String("inr"),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name:        stripe.String(item.Name),
					Description: stripe.String(fmt.Sprintf("Item from Restaurant ID: %d", item.RestaurantID)),
					Images:      []*string{stripe.String(fmt.Sprintf("https://storage.cloud.google.com/foodhaven_bucket/Images/%s", item.CloudImageID))},
				},
				UnitAmount: stripe.Int64(int64(item.Price * 100)), // Price in cents
			},
			Quantity: stripe.Int64(int64(item.Quantity)),
		})
	}

	params := &stripe.CheckoutSessionParams{
		UIMode: stripe.String("embedded"),
		ReturnURL: stripe.String(domain + "/return?session_id={CHECKOUT_SESSION_ID}"),
		LineItems: lineItems,
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)), // Use "payment" for one-time purchases.
	}

	s, err := session.New(params)
	if err != nil {
		log.Printf("Error creating checkout session: %v", err)
		WriteError(w, r, http.StatusInternalServerError, "Failed to create checkout session")
		return
	}

	response := struct {
		ClientSecret string `json:"clientSecret"`
	}{
		ClientSecret: s.ClientSecret,
	}

	WriteSuccessMessage(w, r, response)
}

func RetrieveCheckoutSession(w http.ResponseWriter, r *http.Request) {
	log.Println("--------------------Handling Retrieve Request--------------------")
	setupResponse(&w)

	if r.Method != http.MethodGet {
		WriteError(w, r, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	
	log.Printf("R: %v", r)
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		WriteError(w, r, http.StatusBadRequest, "Missing session_id")
		return
	}


	s, err := session.Get(sessionID, nil)
	if err != nil {
		log.Printf("Error retrieving checkout session: %v", err)
		WriteError(w, r, http.StatusInternalServerError, "Failed to retrieve checkout session")
		return
	}
	log.Printf("%+v",s)

	response := struct {
		Status        string `json:"status"`
		CustomerEmail string `json:"customer_email"`
	}{
		Status:        string(s.Status),
		CustomerEmail: string(s.CustomerDetails.Email),
	}

	WriteSuccessMessage(w, r, response)
}
