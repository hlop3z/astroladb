package auth

import (
	"github.com/go-chi/chi/v5"
)

func Routes() chi.Router {
	r := chi.NewRouter()

	// user routes
	r.Get("/user", ListUser)
	r.Post("/user", CreateUser)
	r.Get("/user/{id}", GetUser)
	r.Patch("/user/{id}", UpdateUser)
	r.Delete("/user/{id}", DeleteUser)

	return r
}
