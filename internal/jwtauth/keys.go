package jwtauth

import (
	"os"
	"strings"
)

type Keys struct {
	Current string
	Set     map[string][]byte
}

func Load() Keys {
	cur := os.Getenv("JWT_CURRENT_KID")
	raw := os.Getenv("JWT_KEYS")
	kset := map[string][]byte{}
	if raw != "" {
		parts := strings.Split(raw, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" || !strings.Contains(p, ":") {
				continue
			}
			i := strings.IndexByte(p, ':')
			kid := p[:i]
			sec := p[i+1:]
			kset[kid] = []byte(sec)
		}
	}
	if len(kset) == 0 {
		kset["default"] = []byte(os.Getenv("JWT_SECRET"))
		if cur == "" {
			cur = "default"
		}
	}
	if cur == "" {
		for k := range kset {
			cur = k
			break
		}
	}
	return Keys{Current: cur, Set: kset}
}
