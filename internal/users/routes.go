package users

import "github.com/go-chi/chi/v5"

// Routes returns a router for the users feature. Assumes it will be
// mounted behind the Auth middleware — every route here can rely on
// appmiddleware.UserIDFromContext being populated.
func Routes(h *Handler) chi.Router {
	r := chi.NewRouter()

	r.Route("/me", func(api chi.Router) {
		api.Get("/", h.GetMe)
		api.Put("/profile", h.UpdateProfile)

		api.Route("/addresses", func(addr chi.Router) {
			addr.Get("/", h.ListAddresses)
			addr.Post("/", h.CreateAddress)
			addr.Put("/{id}", h.UpdateAddress)
			addr.Delete("/{id}", h.DeleteAddress)
		})
	})

	return r
}
