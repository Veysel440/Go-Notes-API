package middleware

import (
	"net/http"

	"github.com/Veysel440/go-notes-api/internal/repos"
)

func RequireRole(rr *repos.Roles, roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uid, ok := UserID(r.Context())
			if !ok {
				http.Error(w, "unauthorized", 401)
				return
			}
			for _, role := range roles {
				ok, err := rr.Has(r.Context(), uid, role)
				if err == nil && ok {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, "forbidden", 403)
		})
	}
}
