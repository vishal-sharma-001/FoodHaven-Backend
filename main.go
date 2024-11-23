package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/rs/cors"

	"github.com/vishal-sharma-001/FoodHaven-Backend/middleware"
	"github.com/vishal-sharma-001/FoodHaven-Backend/routes"
)

func main() {
	port := ":8080"
	router := mux.NewRouter().StrictSlash(true)

	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found. Falling back to system environment variables.")
	}

	sessionKey := os.Getenv("SESSION_KEY")
	if sessionKey == "" {
		log.Fatal("SESSION_KEY is missing. Please set it in your environment variables or .env file.")
	}

	store := sessions.NewCookieStore([]byte(sessionKey))
	store.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	}

	publicRoutes := router.PathPrefix("/public").Subrouter()
	routes.RegisterFoodRoutes(publicRoutes)
	routes.RegisterRestaurantsRoutes(publicRoutes)
	routes.RegisterUserRoutes(publicRoutes, store)
	
	protectedRoutes := router.PathPrefix("/private").Subrouter()
	protectedRoutes.Use(middleware.Authenticate(store))
	routes.RegisterProtectedUserRoutes(protectedRoutes, store)

	uiDir := "./FoodHavenUI"
	if _, err := os.Stat(uiDir); os.IsNotExist(err) {
		log.Printf("Warning: UI directory %q not found. Ensure UI files are present for static serving.\n", uiDir)
	} else {
		router.PathPrefix("/").Handler(http.FileServer(http.Dir(uiDir)))
	}

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-CSRF-Token"},
		AllowCredentials: true,
		Debug:            false,
	}).Handler(router)

	log.Printf("Starting the server on http://localhost%v\n", port)
	if err := http.ListenAndServe(port, corsHandler); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
