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
	log.Println("------Inside main--------")
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
		MaxAge:   86400,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}

	// Public routes
	publicRoutes := router.PathPrefix("/public").Subrouter()
	routes.RegisterFoodRoutes(publicRoutes)
	routes.RegisterRestaurantsRoutes(publicRoutes)
	routes.RegisterUserRoutes(publicRoutes, store)

	// Protected routes
	protectedRoutes := router.PathPrefix("/private").Subrouter()
	protectedRoutes.Use(middleware.Authenticate(store))
	routes.RegisterProtectedUserRoutes(protectedRoutes, store)

	// Path to your frontend UI directory
	uiDir := "./FoodHavenUI"

	// Serve static files (CSS, JS, images, etc.)
	if _, err := os.Stat(uiDir); os.IsNotExist(err) {
		log.Printf("Warning: UI directory %q not found. Ensure UI files are present for static serving.\n", uiDir)
	} else {
		router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(uiDir+"/static"))))
	}

	// Catch-all route to serve index.html for SPA routing
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Serving index.html for route:", r.URL.Path)
		http.ServeFile(w, r, uiDir+"/index.html")
	})

	// CORS middleware
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://34.131.110.94:30080"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-CSRF-Token"},
		AllowCredentials: true,
		Debug:            false,
	}).Handler(router)

	// Start the server
	log.Printf("Starting the server on http://localhost%v\n", port)
	if err := http.ListenAndServe(port, corsHandler); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
