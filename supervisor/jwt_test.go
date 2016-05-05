package supervisor_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/markbates/goth/gothic"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/starkandwayne/shield/supervisor"
)

var _ = Describe("JWTCreators", func() {
	var jc JWTCreator
	var validateToken = func(token string, user string, membership interface{}) {
		parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("Unexpected signing method")
			}
			return jc.SigningKey.Public(), nil
		})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(parsed.Claims["expiration"]).Should(BeNumerically("~", time.Now().Unix()+60, 2.0))
		Expect(parsed.Claims["user"]).Should(Equal(user))
		Expect(parsed.Claims["membership"]).Should(Equal(membership))
	}

	BeforeEach(func() {
		gothic.Store = &FakeSessionStore{}
		data, err := ioutil.ReadFile("test/etc/jwt/valid.pem")
		if err != nil {
			panic(err)
		}
		sk, err := jwt.ParseRSAPrivateKeyFromPEM(data)
		if err != nil {
			panic(err)
		}
		jc.SigningKey = sk
	})

	Describe("When Generating tokens", func() {
		It("Returns a valid token, with all the desired claims set", func() {
			token, err := jc.GenToken("user1", map[string]interface{}{"Groups": []interface{}{"group1", "group2"}}, 60+int(time.Now().Unix()))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(token).ShouldNot(Equal(""))

			validateToken(token, "user1", map[string]interface{}{"Groups": []interface{}{"group1", "group2"}})
		})
	})
	Describe("When Serving Requests", func() {
		var req *http.Request
		var res *FakeResponder
		BeforeEach(func() {
			var err error
			req, err = http.NewRequest("GET", "http://localhost", nil)
			if err != nil {
				panic(err)
			}
			res = NewFakeResponder()
		})
		It("401s if it can't get session info", func() {
			gothic.Store.(*FakeSessionStore).Error = true
			jc.ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(401))
			data, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(data).Should(Equal("No session detected, cannot generate JWT token"))
		})
		It("Returns a token as the response if successful", func() {
			jc.ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(200))
			data, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(data).ShouldNot(Equal(""))

			token := strings.Split(data, " ")
			Expect(len(token)).Should(Equal(2))
			validateToken(token[1], "fakeUser", map[string]interface{}{"Groups": []interface{}{"fakeGroup1", "fakeGroup2"}})
		})
	})
})
