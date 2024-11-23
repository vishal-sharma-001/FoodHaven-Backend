package middleware

import (
	"context"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/vishal-sharma-001/FoodHaven-Backend/database"
	"github.com/vishal-sharma-001/FoodHaven-Backend/models"
)

type contextKey string

const ContextKeyUser = contextKey("user")

func Authenticate(store *sessions.CookieStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := store.Get(r, "user_session")
			if err != nil {
				log.Println("Session error:", err)
				http.Error(w, "Unauthorized: Invalid session", http.StatusUnauthorized)
				return
			}

			userId, ok := session.Values["userId"].(int)
			if !ok || userId == 0 {
				log.Println("Unauthorized: Missing or invalid userId in session")
				http.Error(w, "Unauthorized: User not authenticated", http.StatusUnauthorized)
				return
			}

			user, err := FindUser(userId)
			if err != nil {
				log.Println("Database error:", err)
				http.Error(w, "Unauthorized: User not found", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ContextKeyUser, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func FindUser(userId int) (models.User, error) {
	var user models.User
	dbClient, err := database.ConnectDB()
	if err != nil {
		return user, err
	}
	defer dbClient.Close()

	query := "SELECT id, name, email, password, phone FROM users WHERE id = $1"
	err = dbClient.QueryRow(query, userId).Scan(&user.Id, &user.Name, &user.Email, &user.Password, &user.Phone)
	if err != nil {
		log.Printf("Error fetching user with ID %d: %v", userId, err)
		return user, err
	}
	return user, nil
}
