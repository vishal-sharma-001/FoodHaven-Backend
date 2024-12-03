package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	db "github.com/vishal-sharma-001/FoodHaven-Backend/database"
	"github.com/vishal-sharma-001/FoodHaven-Backend/middleware"
	"github.com/vishal-sharma-001/FoodHaven-Backend/models"

	cashfree "github.com/cashfree/cashfree-pg/v3"
)

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

func CreateOrderHandler(w http.ResponseWriter, r *http.Request) {
	setupResponse(&w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		WriteError(w, r, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	clientID := os.Getenv("PAYMENT_CLIENT_ID")
	secretKey := os.Getenv("PAYMENT_SECRET_KEY")
	if clientID == "" || secretKey == "" {
		log.Println("PAYMENT_CLIENT_ID or PAYMENT_SECRET_KEY is missing. Please set them in your environment variables.")
		WriteError(w, r, http.StatusInternalServerError, "Payment configuration error")
		return
	}

	user, ok := r.Context().Value(middleware.ContextKeyUser).(models.User)
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "User not found in context")
		return
	}

	var order models.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		WriteError(w, r, http.StatusBadRequest, "Invalid request payload")
		return
	}

	cashfree.XClientId = &clientID
	cashfree.XClientSecret = &secretKey
	cashfree.XEnvironment = cashfree.SANDBOX

	version := time.Now().Format("2006-01-02")

	request := cashfree.CreateOrderRequest{
		OrderAmount: float64(order.TotalAmount),
		CustomerDetails: cashfree.CustomerDetails{
			CustomerId:    strconv.Itoa(user.Id),
			CustomerPhone: user.Phone,
			CustomerEmail: &user.Email,
		},
		OrderCurrency: "INR",
		OrderNote: func() *string {
			note := "Order for FoodHaven"
			return &note
		}(),
	}

	response, httpResponse, err := cashfree.PGCreateOrder(&version, &request, nil, nil, nil)
	if err != nil {
		log.Printf("Cashfree order creation failed: %v", err)
		WriteError(w, r, http.StatusInternalServerError, "Failed to create payment order")
		return
	}

	if httpResponse.StatusCode != http.StatusOK {
		WriteError(w, r, httpResponse.StatusCode, "Payment gateway error")
		return
	}

	dbClient, err := db.ConnectDB()
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Database connection error")
		return
	}
	defer dbClient.Close()

	err = dbClient.QueryRow(`
		INSERT INTO orders (user_id, items, total_amount, currency, status) 
		VALUES ($1, $2, $3, $4, 'pending') 
		RETURNING id`,
		user.Id, order.Items, order.TotalAmount, order.Currency).Scan(&order.ID)
	if err != nil {
		log.Printf("Database insertion failed: %v", err)
		WriteError(w, r, http.StatusInternalServerError, "Failed to save order")
		return
	}

	WriteSuccessMessage(w, r, map[string]interface{}{
		"orderID":          order.ID,
		"paymentSessionId": response.PaymentSessionId,
		"orderStatus":      response.OrderStatus,
		"createdAt":        response.CreatedAt,
	})
}

func CheckOrderStatusHandler(w http.ResponseWriter, r *http.Request) {
	setupResponse(&w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodGet {
		WriteError(w, r, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	// Extract order ID from query parameters
	orderID := r.URL.Query().Get("order_id")
	if orderID == "" {
		WriteError(w, r, http.StatusBadRequest, "Order ID is required")
		return
	}

	// Get client credentials from environment variables
	clientID := os.Getenv("PAYMENT_CLIENT_ID")
	secretKey := os.Getenv("PAYMENT_SECRET_KEY")
	if clientID == "" || secretKey == "" {
		log.Println("PAYMENT_CLIENT_ID or PAYMENT_SECRET_KEY is missing.")
		WriteError(w, r, http.StatusInternalServerError, "Payment configuration error")
		return
	}

	// Call the Cashfree API to check the payment status
	url := fmt.Sprintf("https://sandbox.cashfree.com/pg/orders/%s", orderID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		WriteError(w, r, http.StatusInternalServerError, "Failed to check payment status")
		return
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-api-version", "2023-08-01")
	req.SetBasicAuth(clientID, secretKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error checking payment status: %v", err)
		WriteError(w, r, http.StatusInternalServerError, "Failed to fetch payment status")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error checking payment status: HTTP %d", resp.StatusCode)
		WriteError(w, r, resp.StatusCode, "Payment gateway returned an error")
		return
	}

	// Parse the API response
	var responseBody map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		log.Printf("Failed to parse response: %v", err)
		WriteError(w, r, http.StatusInternalServerError, "Invalid payment gateway response")
		return
	}

	// Check payment status
	orderStatus, ok := responseBody["order_status"].(string)
	if !ok || orderStatus != "PAID" {
		WriteError(w, r, http.StatusBadRequest, "Payment not completed or invalid response")
		return
	}

	// Update the order status in the database
	dbClient, err := db.ConnectDB()
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Database connection error")
		return
	}
	defer dbClient.Close()

	query := `UPDATE orders SET status = 'completed', payment_id = $1, updated_at = NOW() WHERE id = $2`
	_, err = dbClient.Exec(query, responseBody["cf_payment_id"], orderID)
	if err != nil {
		if err == sql.ErrNoRows {
			WriteError(w, r, http.StatusNotFound, "Order not found or does not belong to user")
			return
		}
		log.Printf("Database update failed: %v", err)
		WriteError(w, r, http.StatusInternalServerError, "Failed to update order")
		return
	}

	// Send success response
	WriteSuccessMessage(w, r, map[string]interface{}{
		"message":    "Order completed successfully",
		"order_id":   orderID,
		"payment_id": responseBody["cf_payment_id"],
		"status":     orderStatus,
	})
}
