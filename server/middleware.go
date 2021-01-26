package server

import (
	"crypto/rsa"
	"fmt"
	"net/http"

	"github.com/auth0/go-jwt-middleware"
	"github.com/form3tech-oss/jwt-go"
	"golang.org/x/crypto/ssh"
)

func CharmIdFromRequest(r *http.Request) (string, error) {
	user := r.Context().Value("user")
	if user == "" {
		return "", fmt.Errorf("missing user key in context")
	}
	cl := user.(*jwt.Token).Claims.(jwt.MapClaims)
	id, ok := cl["sub"]
	if !ok {
		return "", fmt.Errorf("missing user key in claims map")
	}
	return id.(string), nil
}

func signRSA(pk *rsa.PublicKey) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		return pk, nil
	}
}

func JWTMiddleware(publicKey []byte) (func(http.Handler) http.Handler, error) {
	parsed, _, _, _, err := ssh.ParseAuthorizedKey(publicKey)
	if err != nil {
		return nil, err
	}
	parsedCryptoKey := parsed.(ssh.CryptoPublicKey)
	pubCrypto := parsedCryptoKey.CryptoPublicKey()
	pk, ok := pubCrypto.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("Invalid key")
	}
	return jwtmiddleware.New(jwtmiddleware.Options{
		ValidationKeyGetter: signRSA(pk),
		SigningMethod:       jwt.SigningMethodRS512,
	}).Handler, nil
}
