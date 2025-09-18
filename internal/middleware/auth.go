package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/Veysel440/go-notes-api/internal/jwtauth"
	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const userKey ctxKey = "uid"

func UserID(ctx context.Context) (int64, bool) {
	v := ctx.Value(userKey)
	id, ok := v.(int64)
	return id, ok
}

func Auth() func(http.Handler) http.Handler {
	keys := jwtauth.Load()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				http.Error(w, "unauthorized", 401)
				return
			}
			tokStr := strings.TrimPrefix(h, "Bearer ")
			tok, err := jwt.Parse(tokStr, func(t *jwt.Token) (interface{}, error) {
				if kid, _ := t.Header["kid"].(string); kid != "" {
					if k, ok := keys.Set[kid]; ok {
						return k, nil
					}
				}
				for _, k := range keys.Set {
					return k, nil
				}
				return nil, jwt.ErrTokenMalformed
			})
			if err != nil || !tok.Valid {
				http.Error(w, "unauthorized", 401)
				return
			}
			claims, _ := tok.Claims.(jwt.MapClaims)
			if exp, ok := claims["exp"].(float64); ok && time.Now().Unix() > int64(exp) {
				http.Error(w, "expired", 401)
				return
			}
			idf, ok := claims["sub"].(float64)
			if !ok {
				http.Error(w, "unauthorized", 401)
				return
			}
			ctx := context.WithValue(r.Context(), userKey, int64(idf))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
