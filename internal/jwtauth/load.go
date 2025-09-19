package jwtauth

import (
	"os"
	"strings"
)

var provider KeyProvider = loadFromEnv()

func SetProvider(p KeyProvider)        { provider = p }
func CurrentKID() string               { return provider.CurrentKID() }
func Secret(kid string) ([]byte, bool) { return provider.SecretFor(kid) }

func loadFromEnv() KeyProvider {
	keys := strings.TrimSpace(os.Getenv("JWT_KEYS"))
	current := strings.TrimSpace(os.Getenv("JWT_CURRENT_KID"))
	secret := os.Getenv("JWT_SECRET")

	set := map[string][]byte{}
	if keys != "" {
		for _, p := range strings.Split(keys, ",") {
			kv := strings.SplitN(strings.TrimSpace(p), ":", 2)
			if len(kv) == 2 {
				set[kv[0]] = []byte(kv[1])
			}
		}
	}
	if len(set) == 0 && secret != "" { // fallback
		if current == "" {
			current = "key1"
		}
		set[current] = []byte(secret)
	}
	return EnvProvider{Current: current, Set: set}
}
