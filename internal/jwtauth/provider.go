package jwtauth

type KeyProvider interface {
	CurrentKID() string
	SecretFor(kid string) ([]byte, bool)
}

type EnvProvider struct {
	Current string
	Set     map[string][]byte
}

func (e EnvProvider) CurrentKID() string                  { return e.Current }
func (e EnvProvider) SecretFor(kid string) ([]byte, bool) { v, ok := e.Set[kid]; return v, ok }
