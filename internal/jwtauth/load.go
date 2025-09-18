package jwtauth

import (
	"os"
	"strings"
)

type KeyProvider interface {
	CurrentKID() string
	SecretFor(kid string) ([]byte, bool)
}

type EnvProvider struct {
	Current string
	Set     map[string][]byte
}

func (e EnvProvider) CurrentKID() string { return e.Current }
func (e EnvProvider) SecretFor(kid string) ([]byte, bool) {
	v, ok := e.Set[kid]
	return v, ok
}

var provider KeyProvider = loadFromEnv()

func SetProvider(p KeyProvider) { provider = p }
func CurrentKID() string        { return provider.CurrentKID() }
func Secret(kid string) ([]byte, bool) {
	return provider.SecretFor(kid)
}

func loadFromEnv() KeyProvider {
	// JWT_KEYS="kid1:secret1,kid2:secret2"
	keys := strings.TrimSpace(os.Getenv("JWT_KEYS"))
	current := strings.TrimSpace(os.Getenv("JWT_CURRENT_KID"))
	secret := os.Getenv("JWT_SECRET")

	set := map[string][]byte{}
	if keys != "" {
		pairs := strings.Split(keys, ",")
		for _, p := range pairs {
			kv := strings.SplitN(strings.TrimSpace(p), ":", 2)
			if len(kv) == 2 {
				set[kv[0]] = []byte(kv[1])
			}
		}
	}
	// tek secret fallback
	if len(set) == 0 && secret != "" {
		if current == "" {
			current = "key1"
		}
		set[current] = []byte(secret)
	}
	return EnvProvider{Current: current, Set: set}
}
