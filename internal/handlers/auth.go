package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Veysel440/go-notes-api/internal/config"
	"github.com/Veysel440/go-notes-api/internal/jwtauth"
	"github.com/Veysel440/go-notes-api/internal/repos"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Auth struct {
	Cfg          config.Config
	Users        *repos.Users
	Tokens       *repos.RefreshTokens
	Roles        *repos.Roles
	EmailLimiter func(string) bool
	Metrics      *repos.AuthMetrics
}

type creds struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

var validate = validator.New(validator.WithRequiredStructEnabled())

func randID() string { var b [16]byte; _, _ = rand.Read(b[:]); return hex.EncodeToString(b[:]) }

func (h Auth) Register(w http.ResponseWriter, r *http.Request) {
	var in creds
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	var v struct {
		Email    string `validate:"required,email,max=200"`
		Password string `validate:"required,min=8,max=128"`
	}
	v.Email, v.Password = in.Email, in.Password
	if err := validate.Struct(v); err != nil {
		http.Error(w, "invalid", 422)
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(in.Password), h.Cfg.BcryptCost)

	ctx, cancel := context.WithTimeout(r.Context(), h.Cfg.DBTimeout)
	defer cancel()

	id, err := h.Users.Create(ctx, in.Email, string(hash))
	if err != nil {
		http.Error(w, "conflict", 409)
		return
	}
	if h.Roles != nil {
		_ = h.Roles.Assign(ctx, id, "user")
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"id": id})
}

func (h Auth) Login(w http.ResponseWriter, r *http.Request) {
	var in creds
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	if h.EmailLimiter != nil && !h.EmailLimiter(in.Email) {
		http.Error(w, "rate limit", 429)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.Cfg.DBTimeout)
	defer cancel()

	u, err := h.Users.FindByEmail(ctx, in.Email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)) != nil {
		if h.Metrics != nil {
			h.Metrics.Failed.Inc()
		}
		time.Sleep(250 * time.Millisecond)
		http.Error(w, "unauthorized", 401)
		return
	}

	keys := jwtauth.Load()
	claims := jwt.MapClaims{
		"sub": u.ID,
		"exp": time.Now().Add(h.Cfg.JWTTTL).Unix(),
		"iat": time.Now().Unix(),
		"iss": h.Cfg.JWTIssuer,
		"aud": h.Cfg.JWTAudience,
		"jti": randID(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t.Header["kid"] = keys.Current
	access, _ := t.SignedString(keys.Set[keys.Current])

	rt, err := h.Tokens.Issue(ctx, u.ID, time.Now().Add(h.Cfg.RefreshTTL))
	if err != nil {
		http.Error(w, "server", 500)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"access": access, "refresh": rt})
}

func (h Auth) Refresh(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Refresh string `json:"refresh"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad json", 400)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.Cfg.DBTimeout)
	defer cancel()

	uid, newRT, reused, err := h.Tokens.UseAndRotate(ctx, in.Refresh, time.Now().Add(h.Cfg.RefreshTTL))
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "invalid", 401)
			return
		}
		http.Error(w, "server", 500)
		return
	}
	if reused {
		http.Error(w, "token_reused_detected", 401)
		return
	}

	keys := jwtauth.Load()
	claims := jwt.MapClaims{
		"sub": uid,
		"exp": time.Now().Add(h.Cfg.JWTTTL).Unix(),
		"iat": time.Now().Unix(),
		"iss": h.Cfg.JWTIssuer,
		"aud": h.Cfg.JWTAudience,
		"jti": randID(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t.Header["kid"] = keys.Current
	access, _ := t.SignedString(keys.Set[keys.Current])

	_ = json.NewEncoder(w).Encode(map[string]any{"access": access, "refresh": newRT})
}
