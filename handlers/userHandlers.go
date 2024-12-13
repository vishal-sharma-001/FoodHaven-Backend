package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

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

func FetchCart(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodGet {
		WriteError(w, r, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	// Get the authenticated user from the context
	user, ok := r.Context().Value(middleware.ContextKeyUser).(models.User)
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Connect to the database
	db, err := db.ConnectDB()
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Database connection error")
		return
	}
	defer db.Close()

	var cartID int
	var restaurantID sql.NullInt64 // to handle nullable restaurant_id

	// Fetch the cart ID and restaurant_id (if available)
	err = db.QueryRow("SELECT id, restaurant_id FROM cart WHERE user_id = $1 AND is_active = TRUE", user.Id).Scan(&cartID, &restaurantID)

	if err != nil {
		if err == sql.ErrNoRows {
			// No active cart found, create a new one with a default restaurant_id (NULL allowed)
			log.Printf("No active cart found, creating a new one")

			err = db.QueryRow("INSERT INTO cart (user_id, total_amount, is_active) VALUES ($1, 0, TRUE) RETURNING id", user.Id).Scan(&cartID)

			if err != nil {
				WriteError(w, r, http.StatusInternalServerError, "Failed to create new cart")
				return
			}

			// Return the new cart with no items
			response := map[string]interface{}{
				"cart_id": cartID,
				"items":   []models.OrderItem{},
			}
			WriteSuccessMessage(w, r, response)
			return
		}

		// Handle other database errors
		WriteError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// Fetch items for the active cart
	rows, err := db.Query(`
		SELECT 
			ci.item_id, 
			fi.name, 
			ci.quantity,
			fi.price::numeric, -- Cast price to NUMERIC to avoid type issues
			fi.cloudimageid
		FROM cart_items ci
		JOIN FoodItems fi ON ci.item_id = fi.id 
		WHERE ci.cart_id = $1`, cartID)

	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	items := []models.OrderItem{}
	for rows.Next() {
		var item models.OrderItem

		// Scan cart item details
		if err := rows.Scan(&item.ID, &item.Name, &item.Quantity, &item.Price, &item.CloudImageID); err != nil {
			WriteError(w, r, http.StatusInternalServerError, err)
			return
		}

		// If restaurant_id is available in the cart, set it for the item
		if restaurantID.Valid {
			item.RestaurantID = int(restaurantID.Int64)
		}

		items = append(items, item)
	}

	// Return the active cart with items
	response := map[string]interface{}{
		"cart_id": cartID,
		"items":   items,
		"restaurantid": int(restaurantID.Int64),
	}

	WriteSuccessMessage(w, r, response)
}

func SyncCart(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		WriteError(w, r, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	// Parse the items from the nested request body
	var payload struct {
		Items []models.OrderItem `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteError(w, r, http.StatusBadRequest, "Invalid request body")
		return
	}
	newItems := payload.Items

	// Extract cart_id from URL
	vars := mux.Vars(r)
	cartID, err := strconv.Atoi(vars["cart_id"])
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, "Invalid cart ID")
		return
	}

	// Extract authenticated user
	user, ok := r.Context().Value(middleware.ContextKeyUser).(models.User)
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Connect to the database
	db, err := db.ConnectDB()
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Database connection error")
		return
	}
	defer db.Close()

	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
			log.Printf("Transaction rollback due to error: %v", err)
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				log.Printf("Transaction commit failed: %v", commitErr)
			}
		}
	}()

	// Verify that the cart belongs to the user and is active
	var dbUserID int
	err = tx.QueryRow("SELECT user_id FROM cart WHERE id = $1 AND is_active = TRUE", cartID).Scan(&dbUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			WriteError(w, r, http.StatusNotFound, "Cart not found or inactive")
		} else {
			WriteError(w, r, http.StatusInternalServerError, "Failed to fetch cart")
		}
		return
	}
	if dbUserID != user.Id {
		WriteError(w, r, http.StatusForbidden, "Unauthorized to modify this cart")
		return
	}

	// Fetch existing items from the database
	existingItems := make(map[int]models.OrderItem)
	rows, err := tx.Query("SELECT item_id, quantity FROM cart_items WHERE cart_id = $1", cartID)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Failed to fetch existing cart items")
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.OrderItem
		if err := rows.Scan(&item.ID, &item.Quantity); err != nil {
			WriteError(w, r, http.StatusInternalServerError, "Failed to parse existing cart items")
			return
		}
		existingItems[item.ID] = item
	}

	// Synchronize items
	totalAmount := 0.0
	var restaurantID int
	updatedItems := make(map[int]bool)

	for _, newItem := range newItems {
		if restaurantID == 0 {
			restaurantID = newItem.RestaurantID
		} else if restaurantID != newItem.RestaurantID {
			WriteError(w, r, http.StatusBadRequest, "All items must be from the same restaurant")
			return
		}

		existingItem, exists := existingItems[newItem.ID]
		if exists {
			// Update the quantity if it differs
			if existingItem.Quantity != newItem.Quantity {
				_, err = tx.Exec(
					"UPDATE cart_items SET quantity = $1, updated_at = NOW() WHERE cart_id = $2 AND item_id = $3",
					newItem.Quantity, cartID, newItem.ID,
				)
				if err != nil {
					WriteError(w, r, http.StatusInternalServerError, "Failed to update cart item")
					return
				}
			}
		} else {
			// Insert new item
			_, err = tx.Exec(
				"INSERT INTO cart_items (cart_id, item_id, quantity, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())",
				cartID, newItem.ID, newItem.Quantity,
			)
			if err != nil {
				WriteError(w, r, http.StatusInternalServerError, "Failed to insert cart item")
				return
			}
		}
		totalAmount += float64(newItem.Quantity) * float64(newItem.Price)
		updatedItems[newItem.ID] = true
	}

	// Remove items that are no longer in the cart
	for existingID := range existingItems {
		if !updatedItems[existingID] {
			_, err = tx.Exec("DELETE FROM cart_items WHERE cart_id = $1 AND item_id = $2", cartID, existingID)
			if err != nil {
				WriteError(w, r, http.StatusInternalServerError, "Failed to delete old cart item")
				return
			}
		}
	}

	// Update the cart with the total amount and restaurant ID
	_, err = tx.Exec(
		"UPDATE cart SET total_amount = $1, restaurant_id = $2, updated_at = NOW() WHERE id = $3",
		totalAmount, restaurantID, cartID,
	)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Failed to update cart")
		return
	}

	log.Printf("Cart %d synced successfully for user %d", cartID, user.Id)
	WriteSuccessMessage(w, r, "Sync Successful")
}

func CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("Error decoding request body: %v", err)
		WriteError(w, r, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, ok := r.Context().Value(middleware.ContextKeyUser).(models.User)
	if !ok {
		log.Printf("User not found in context")
		WriteError(w, r, http.StatusUnauthorized, "Unauthorized: User not found")
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
				UnitAmount: stripe.Int64(int64(item.Price * 100)),
			},
			Quantity: stripe.Int64(int64(item.Quantity)),
		})
	}

	params := &stripe.CheckoutSessionParams{
		UIMode:    stripe.String("embedded"),
		ReturnURL: stripe.String(domain + "/return?session_id={CHECKOUT_SESSION_ID}"),
		LineItems: lineItems,
		Mode:      stripe.String(string(stripe.CheckoutSessionModePayment)),
	}

	s, err := session.New(params)
	if err != nil {
		log.Printf("Error creating Stripe session: %v", err)
		WriteError(w, r, http.StatusInternalServerError, "Failed to create checkout session")
		return
	}

	dbClient, err := db.ConnectDB()
	if err != nil {
		log.Printf("Database connection error: %v", err)
		WriteError(w, r, http.StatusInternalServerError, "Database connection error")
		return
	}
	defer dbClient.Close()

	// Start a transaction
	tx, err := dbClient.Begin()
	if err != nil {
		log.Printf("Failed to start database transaction: %v", err)
		WriteError(w, r, http.StatusInternalServerError, "Transaction initialization error")
		return
	}

	// Use defer to handle transaction commit/rollback
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			log.Printf("Panic occurred: %v. Transaction rolled back.", p)
			panic(p)
		} else if err != nil {
			tx.Rollback()
			log.Printf("Transaction rolled back due to error: %v", err)
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				log.Printf("Failed to commit transaction: %v", commitErr)
				WriteError(w, r, http.StatusInternalServerError, "Transaction commit error")
			}
		}
	}()
     
	log.Printf("Request: %+v", req)
	// Insert into orders table (payment_id is NULL)
	var orderID int
	err = tx.QueryRow(`
		INSERT INTO orders (user_id, session_id, total_amount, currency, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING order_id`,
		user.Id, s.ID, req.Amount, "INR", "pending",
	).Scan(&orderID)
	if err != nil {
		log.Printf("Error inserting order: %v", err)
		WriteError(w, r, http.StatusInternalServerError, "Failed to save order")
		return
	}

	// Insert items into order_items table
	for _, item := range req.Items {
		_, err = tx.Exec(`
			INSERT INTO order_items (order_id, item_id, quantity, price, created_at)
			VALUES ($1, $2, $3, $4, NOW())`,
			orderID, item.ID, item.Quantity, item.Price,
		)
		if err != nil {
			log.Printf("Error inserting order items for order ID %d: %v", orderID, err)
			WriteError(w, r, http.StatusInternalServerError, "Failed to save order items")
			return
		}
	}

	// Return response
	response := struct {
		ClientSecret string `json:"clientSecret"`
		OrderID      int    `json:"orderId"`
	}{
		ClientSecret: s.ClientSecret,
		OrderID:      orderID,
	}

	WriteSuccessMessage(w, r, response)
}


func RetrieveCheckoutSession(w http.ResponseWriter, r *http.Request) {
    setupResponse(&w)

    if r.Method != http.MethodGet {
        WriteError(w, r, http.StatusMethodNotAllowed, "Invalid request method")
        return
    }

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

    dbClient, err := db.ConnectDB()
    if err != nil {
        log.Printf("Database connection error: %v", err)
        WriteError(w, r, http.StatusInternalServerError, "Database connection error")
        return
    }
    defer dbClient.Close()

    // Determine the status to update in the orders table
    var orderStatus string
    switch s.Status {
    case stripe.CheckoutSessionStatusComplete:
        orderStatus = "completed"
    case stripe.CheckoutSessionStatusExpired:
        orderStatus = "expired"
    default:
        orderStatus = "failed"
    }

    // Update the order in the database
    _, err = dbClient.Exec(
        `UPDATE orders SET status = $1, payment_id = $2, updated_at = NOW() WHERE session_id = $3`,
        orderStatus, s.ID, sessionID,
    )
    if err != nil {
        log.Printf("Failed to update order for session %s: %v", sessionID, err)
        WriteError(w, r, http.StatusInternalServerError, "Failed to update order status")
        return
    }

	// Extract authenticated user
	user, ok := r.Context().Value(middleware.ContextKeyUser).(models.User)
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "User not found in context")
		return
	}

    // Update the cart to set is_active = false
    _, err = dbClient.Exec(`UPDATE cart SET is_active = false WHERE user_id = $1`, user.Id)
    if err != nil {
        log.Printf("Failed to update cart for user %v", err)
        WriteError(w, r, http.StatusInternalServerError, "Failed to update cart status")
        return
    }

    // Prepare the response
    response := struct {
        Status        string `json:"status"`
        CustomerEmail string `json:"customer_email"`
    }{
        Status:        string(s.Status),
        CustomerEmail: string(s.CustomerDetails.Email),
    }

    WriteSuccessMessage(w, r, response)
}


func FetchOrders(w http.ResponseWriter, r *http.Request) {
    setupResponse(&w)

    if r.Method != http.MethodGet {
        WriteError(w, r, http.StatusMethodNotAllowed, "Invalid request method")
        return
    }

    // Retrieve authenticated user from context
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

    // Query to fetch orders with aggregated order items and food item details
    rows, err := dbClient.Query(`
        SELECT 
          o.order_id, 
          o.total_amount, 
          o.currency, 
          o.status, 
          o.payment_id, 
          o.created_at, 
          o.updated_at, 
          json_agg(json_build_object(
            'item_id', oi.item_id, 
            'quantity', oi.quantity, 
            'price', oi.price, 
            'name', fi.name, 
            'cloudimageid', fi.cloudimageid
          )) AS items
        FROM orders o
        LEFT JOIN order_items oi ON o.order_id = oi.order_id
        LEFT JOIN FoodItems fi ON oi.item_id = fi.id
        WHERE o.user_id = $1
        GROUP BY o.order_id
        ORDER BY o.created_at DESC`, user.Id)

    if err != nil {
        log.Printf("Error fetching orders: %v", err)
        WriteError(w, r, http.StatusInternalServerError, "Failed to fetch orders")
        return
    }
    defer rows.Close()

    orders := []map[string]interface{}{}

    // Parse rows to build response
    for rows.Next() {
        var orderID int
        var totalAmount float64
        var currency, status, paymentID sql.NullString
        var createdAt, updatedAt time.Time
        var itemsJSON sql.NullString

        if err := rows.Scan(&orderID, &totalAmount, &currency, &status, &paymentID, &createdAt, &updatedAt, &itemsJSON); err != nil {
            log.Printf("Error scanning order row: %v", err)
            WriteError(w, r, http.StatusInternalServerError, "Failed to parse order data")
            return
        }

        // Parse items JSON if not null
        var items []map[string]interface{}
        if itemsJSON.Valid {
            if err := json.Unmarshal([]byte(itemsJSON.String), &items); err != nil {
                log.Printf("Error unmarshaling items JSON: %v", err)
                WriteError(w, r, http.StatusInternalServerError, "Failed to parse order items")
                return
            }
        }

        // Build order object
        order := map[string]interface{}{
            "order_id":    orderID,
            "total_amount": totalAmount,
            "currency":    currency.String,
            "status":      status.String,
            "payment_id":  paymentID.String,
            "created_at":  createdAt,
            "updated_at":  updatedAt,
            "items":       items,
        }

        orders = append(orders, order)
    }

    if err := rows.Err(); err != nil {
        log.Printf("Error iterating order rows: %v", err)
        WriteError(w, r, http.StatusInternalServerError, "Error fetching orders")
        return
    }

    // Respond with the list of orders
    WriteSuccessMessage(w, r, orders)
}
