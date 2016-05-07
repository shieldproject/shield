package supervisor

import (
	"crypto/rsa"
	"github.com/dgrijalva/jwt-go"
	"github.com/markbates/goth/gothic"
	"net/http"
	"time"
)

type JWTCreator struct {
	SigningKey *rsa.PrivateKey
}

func (jc JWTCreator) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sess, err := gothic.Store.Get(r, gothic.SessionName)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte("No session detected, cannot generate JWT token"))
		return
	}

	token, err := jc.GenToken(sess.Values["User"], sess.Values["Membership"], sess.Options.MaxAge+int(time.Now().Unix()))
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Unable to generate authentication token"))
		return
	}
	w.Write([]byte("Bearer " + token))
}

func (jc JWTCreator) GenToken(user interface{}, membership interface{}, maxAge int) (string, error) {
	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims["expiration"] = maxAge
	token.Claims["user"] = user
	token.Claims["membership"] = membership

	return token.SignedString(jc.SigningKey)
}
