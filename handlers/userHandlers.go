package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	db "github.com/vishal-sharma-001/FoodHaven-Backend/database"
	"github.com/vishal-sharma-001/FoodHaven-Backend/middleware"
	"github.com/vishal-sharma-001/FoodHaven-Backend/models"
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

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	query := "INSERT INTO users (name, email, phone, password) VALUES ($1, $2, $3, $4) RETURNING id"
	dbClient, err := db.ConnectDB()
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Database connection error")
		return
	}
	defer dbClient.Close()

	var userID int
	err = dbClient.QueryRow(query, user.Name, user.Email, user.Phone, hashedPassword).Scan(&userID)
	if err, ok := err.(*pq.Error); ok && err.Code == "23505" {
		if strings.Contains(err.Message, "users_email_key") {
			WriteError(w, r, http.StatusConflict, "Email is already registered")
		} else if strings.Contains(err.Message, "users_phone_key") {
			WriteError(w, r, http.StatusConflict, "Phone number is already registered")
		} else {
			WriteError(w, r, http.StatusInternalServerError, "Database error")
		}
		return
	} else if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Failed to register user")
		return
	}

	session, err := store.Get(r, "user_session")
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Failed to create session")
		return
	}

	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   24 * 60 * 60,
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	}

	if err := session.Save(r, w); err != nil {
		WriteError(w, r, http.StatusInternalServerError, "Failed to save session")
		return
	}

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
		MaxAge:   24 * 60 * 60, 
		HttpOnly: false,
		Secure:   true, 
		SameSite: http.SameSiteNoneMode,
	}

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

	rows, err := dbClient.Query(`
        SELECT id, user_id, name, street, city, postal_code, phone, is_primary
        FROM addresses
        WHERE user_id = $1`, user.Id)

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
