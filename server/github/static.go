package github

import (
	"crypto/subtle"
	"errors"
)

type staticAuthenticator struct {
	token string
}

func NewStatic(token string) Authenticator {
	return staticAuthenticator{token: token}
}

func (s staticAuthenticator) OAuthUrl() string { return "" }

func (s staticAuthenticator) ObtainToken(string) (string, error) {
	return "", errors.New("OAuth is disabled")
}

func (s staticAuthenticator) Authenticate(token string) (User, error) {
	if subtle.ConstantTimeCompare([]byte(token), []byte(s.token)) != 1 {
		return User{}, errors.New("invalid token")
	}
	return User{Login: "selfhost", Allowed: true}, nil
}
