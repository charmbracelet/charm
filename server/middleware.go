package server

import (
	"context"
	"crypto/rsa"
	"fmt"
	"log"
	"net/http"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	charm "github.com/charmbracelet/charm/proto"
	"github.com/form3tech-oss/jwt-go"
	"golang.org/x/crypto/ssh"
)

const ctxUserKey = "charmUser"

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

func CharmUserMiddleware(s *HTTPServer) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, err := charmIdFromRequest(r)
			if err != nil {
				log.Printf("cannot get charm id from request: %s", err)
				s.renderError(w)
				return
			}
			u, err := s.db.GetUserWithID(id)
			if err == charm.ErrMissingUser {
				s.renderCustomError(w, fmt.Sprintf("missing user for id '%s'", id), http.StatusNotFound)
				return
			} else if err != nil {
				log.Printf("cannot read request body: %s", err)
				s.renderError(w)
				return
			}
			ctx := context.WithValue(r.Context(), ctxUserKey, u)
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func charmIdFromRequest(r *http.Request) (string, error) {
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
