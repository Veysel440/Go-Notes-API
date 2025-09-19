package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Veysel440/go-notes-api/internal/config"
	apperr "github.com/Veysel440/go-notes-api/internal/errors"
	"github.com/Veysel440/go-notes-api/internal/jwtauth"
	"github.com/Veysel440/go-notes-api/internal/repos"
	"github.com/Veysel440/go-notes-api/internal/security"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type Auth struct {
	Cfg          config.Config
	Users        *repos.Users
	Tokens       *repos.RefreshTokens
	Roles        *repos.Roles
	EmailLimiter func(string) bool
	Metrics      *repos.AuthMetrics
	JTIStore     interface {
		Revoke(ctx context.Context, jti string, ttl time.Duration) error
	}
	BruteRedis *redis.Client
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
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	var v struct {
		Email    string `validate:"required,email,max=200"`
		Password string `validate:"required,min=8,max=128"`
	}
	v.Email, v.Password = in.Email, in.Password
	if err := validate.Struct(v); err != nil {
		http.Error(w, "invalid", http.StatusUnprocessableEntity)
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(in.Password), h.Cfg.BcryptCost)

	ctx, cancel := context.WithTimeout(r.Context(), h.Cfg.DBTimeout)
	defer cancel()

	id, err := h.Users.Create(ctx, in.Email, string(hash))
	if err != nil {
		http.Error(w, "conflict", http.StatusConflict)
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
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if h.EmailLimiter != nil && !h.EmailLimiter(in.Email) {
		http.Error(w, "rate limit", http.StatusTooManyRequests)
		return
	}
	br := security.Brute{RDB: h.BruteRedis, Limit: 10, Window: 5 * time.Minute}
	if ok, _, ttl := br.Allow(r.Context(), r, in.Email); !ok {
		w.Header().Set("Retry-After", strconv.Itoa(int(ttl.Seconds())))
		apperr.Write(w, r, apperr.TooMany)
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
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	claims := jwt.MapClaims{
		"sub": u.ID,
		"exp": time.Now().Add(h.Cfg.JWTTTL).Unix(),
		"iat": time.Now().Unix(),
		"iss": h.Cfg.JWTIssuer,
		"aud": h.Cfg.JWTAudience,
		"jti": randID(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	kid := jwtauth.CurrentKID()
	sec, _ := jwtauth.Secret(kid)
	t.Header["kid"] = kid
	access, _ := t.SignedString(sec)

	rt, err := h.Tokens.Issue(ctx, u.ID, time.Now().Add(h.Cfg.RefreshTTL))
	if err != nil {
		http.Error(w, "server", http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"access": access, "refresh": rt})
}

func (h Auth) Refresh(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Refresh string `json:"refresh"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.Cfg.DBTimeout)
	defer cancel()

	uid, newRT, reused, err := h.Tokens.UseAndRotate(ctx, in.Refresh, time.Now().Add(h.Cfg.RefreshTTL))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "invalid", http.StatusUnauthorized)
			return
		}
		http.Error(w, "server", http.StatusInternalServerError)
		return
	}
	if reused {
		http.Error(w, "token_reused_detected", http.StatusUnauthorized)
		return
	}

	claims := jwt.MapClaims{
		"sub": uid,
		"exp": time.Now().Add(h.Cfg.JWTTTL).Unix(),
		"iat": time.Now().Unix(),
		"iss": h.Cfg.JWTIssuer,
		"aud": h.Cfg.JWTAudience,
		"jti": randID(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	kid := jwtauth.CurrentKID()
	sec, _ := jwtauth.Secret(kid)
	t.Header["kid"] = kid
	access, _ := t.SignedString(sec)

	_ = json.NewEncoder(w).Encode(map[string]any{"access": access, "refresh": newRT})
}

func (h Auth) Logout(w http.ResponseWriter, r *http.Request) {
	hdr := r.Header.Get("Authorization")
	if !strings.HasPrefix(hdr, "Bearer ") {
		apperr.Write(w, r, apperr.Unauthorized)
		return
	}
	raw := strings.TrimPrefix(hdr, "Bearer ")

	keyFn := func(t *jwt.Token) (interface{}, error) {
		if kid, _ := t.Header["kid"].(string); kid != "" {
			if sec, ok := jwtauth.Secret(kid); ok {
				return sec, nil
			}
		}
		if sec, ok := jwtauth.Secret(jwtauth.CurrentKID()); ok {
			return sec, nil
		}
		return nil, jwt.ErrTokenMalformed
	}
	tok, err := jwt.Parse(raw, keyFn)
	if err != nil || !tok.Valid {
		apperr.Write(w, r, apperr.Unauthorized)
		return
	}

	claims, _ := tok.Claims.(jwt.MapClaims)
	jti, _ := claims["jti"].(string)
	exp, _ := claims["exp"].(float64)
	ttl := time.Until(time.Unix(int64(exp), 0))
	if ttl < 0 {
		ttl = 0
	}
	if h.JTIStore != nil && jti != "" {
		_ = h.JTIStore.Revoke(r.Context(), jti, ttl)
	}

	var in struct {
		Refresh string `json:"refresh"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)
	if in.Refresh != "" {
		_ = h.Tokens.Revoke(r.Context(), in.Refresh)
	}
	w.WriteHeader(http.StatusNoContent)
}
