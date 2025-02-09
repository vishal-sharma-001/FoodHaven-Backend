package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	"github.com/vishal-sharma-001/FoodHaven-Backend/middleware"
	"github.com/vishal-sharma-001/FoodHaven-Backend/routes"
)

func main() {
    port := ":8080"
    router := mux.NewRouter().StrictSlash(true)

    sessionKey := os.Getenv("SESSION_KEY")
    if sessionKey == "" {
        log.Fatal("SESSION_KEY is missing. Please set it in your environment variables or .env file.")
    }

    store := sessions.NewCookieStore([]byte(sessionKey))
    store.Options = &sessions.Options{
        Path:     "/",
        MaxAge:   86400,
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteLaxMode,
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
        router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(uiDir+"/static"))))
    }

    router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Println("Serving index.html for route:", r.URL.Path)
        http.ServeFile(w, r, uiDir+"/index.html")
    })

    tlsCertPath := "/etc/tls/tls.crt"
    tlsKeyPath := "/etc/tls/tls.key"

    if _, err := os.Stat(tlsCertPath); os.IsNotExist(err) {
        log.Fatalf("TLS certificate file not found at %v", tlsCertPath)
    }
    if _, err := os.Stat(tlsKeyPath); os.IsNotExist(err) {
        log.Fatalf("TLS key file not found at %v", tlsKeyPath)
    }

    log.Printf("Starting the server on https://localhost%v\n", port)

    server := &http.Server{
        Addr:      port,
        Handler: router,
        TLSConfig: &tls.Config{
            MinVersion: tls.VersionTLS13,
        },
    }

    if err := server.ListenAndServeTLS(tlsCertPath, tlsKeyPath); err != nil {
        log.Fatalf("Error starting server: %v", err)
    }
}