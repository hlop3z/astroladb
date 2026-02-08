// Alab Generated Chi API
//
// Run:
//   go get github.com/go-chi/chi/v5
//   go get github.com/google/uuid
//   go get github.com/shopspring/decimal
//   go run main.go

package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"your-project/auth"
)

func main() {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))

	// Routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"message": "Alab Generated API", "docs": "/docs"}`))
	})

	r.Mount("/auth", auth.Routes())

	fmt.Println("✓ Chi server running on http://localhost:3000")
	http.ListenAndServe(":3000", r)
}
