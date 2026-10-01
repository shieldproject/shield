# Okta JWT Verifier for Golang

This library helps you verify tokens that have been issued by Okta. To learn more about verification cases and Okta's tokens please read [Working With OAuth 2.0 Tokens](https://developer.okta.com/authentication-guide/tokens/)

## Release status

This library uses semantic versioning and follows Okta's [library version policy](https://developer.okta.com/code/library-versions/).

| Version | Status                           |
| ------- | -------------------------------- |
| 0.x     | :warning: Beta Release (Retired) |
| 1.x     | :heavy_check_mark: Release       |

## Installation

```sh
go get -u github.com/okta/okta-jwt-verifier-golang
```

## Usage

This library was built to keep configuration to a minimum. To get it running at its most basic form, all you need to provide is the the following information:

- **Issuer** - This is the URL of the authorization server that will perform authentication. All Developer Accounts have a "default" authorization server. The issuer is a combination of your Org URL (found in the upper right of the console home page) and `/oauth2/default`. For example, `https://dev-1234.oktapreview.com/oauth2/default`.
- **Client ID**- These can be found on the "General" tab of the Web application that you created earlier in the Okta Developer Console.

#### Access Token Validation

```go
import "github.com/okta/okta-jwt-verifier-golang/v2"

toValidate := map[string]string{}
toValidate["aud"] = "api://default"
toValidate["cid"] = "{CLIENT_ID}"

jwtVerifierSetup := jwtverifier.JwtVerifier{
        Issuer: "{ISSUER}",
        ClaimsToValidate: toValidate,
}

verifier := jwtVerifierSetup.New()

token, err := verifier.VerifyAccessToken("{JWT}")
```

#### Id Token Validation

```go
import "github.com/okta/okta-jwt-verifier-golang/v2"

toValidate := map[string]string{}
toValidate["nonce"] = "{NONCE}"
toValidate["aud"] = "{CLIENT_ID}"


jwtVerifierSetup := jwtverifier.JwtVerifier{
        Issuer: "{ISSUER}",
        ClaimsToValidate: toValidate,
}

verifier := jwtVerifierSetup.New()

token, err := verifier.VerifyIdToken("{JWT}")
```

This will either provide you with the token which gives you access to all the claims, or an error. The token struct contains a `Claims` property that will give you a `map[string]interface{}` of all the claims in the token.

```go
// Getting the sub from the token
sub := token.Claims["sub"]
```

#### Dealing with clock skew

We default to a two minute clock skew adjustment in our validation. If you need to change this, you can use the `SetLeeway` method:

```go
jwtVerifierSetup := JwtVerifier{
        Issuer: "{ISSUER}",
}

verifier := jwtVerifierSetup.New()
verifier.SetLeeway("2m") //String instance of time that will be parsed by `time.ParseDuration`
```

#### Customizable Resource Cache

The verifier setup has a default cache based on
[`patrickmn/go-cache`](https://github.com/patrickmn/go-cache) with a 5 minute
expiry and 10 minute purge default setting that is used to store resources fetched over
HTTP. The expiry and purge setting is configurable through SetCleanUp and SetTimeOut method.
It also defines a `Cacher` interface with a `Get` method allowing
customization of that caching. If you want to establish your own caching
strategy then provide your own `Cacher` object that implements that interface.
Your custom cache is set in the verifier via the `Cache` attribute. See the
example in the [cache example test](utils/cache_example_test.go) that shows a
"forever" cache (that one would never use in production ...)

```go
jwtVerifierSetup := jwtverifier.JwtVerifier{
    Cache: NewForeverCache,
    // other fields here
}

verifier := jwtVerifierSetup.New()
```

#### Utilities

The below utilities are available in this package that can be used for Authentication flows

**Nonce Generator**

```go
import jwtUtils "github.com/okta/okta-jwt-verifier-golang/v2/utils"

nonce, err := jwtUtils.GenerateNonce()
```

**PKCE Code Verifier and Challenge Generator**

```go
import jwtUtils "github.com/okta/okta-jwt-verifier-golang/v2/utils"

codeVerifier, err := jwtUtils.GenerateCodeVerifier()
// or
codeVerifier, err := jwtUtils.GenerateCodeVerifierWithLength(50)

// get string value for oauth2 code verifier
codeVerifierValue := codeVerifier.String()

// get plain text code challenge from verifier
codeChallengePlain := codeVerifier.CodeChallengePlain()
codeChallengeMethod := "plain"
// get sha256 code challenge from verifier
codeChallengeS256 := codeVerifier.CodeChallengeS256()
codeChallengeMethod := "S256"
```

## Testing

If you create a PR from a fork of okta/okta-jwt-verifier-golang the build for
the PR will fail. Don't worry, we'll bring your commits into a review branch in
okta/okta-jwt-verifier-golang and get a green build.

jwtverifier_test.go expects environment variables for `ISSUER`, `CLIENT_ID`,
`USERNAME`, and `PASSWORD` to be present. Take note if you use zshell as
`USERSNAME` is a special environment variable and is not settable. Therefore
tests shouldn't be run in zshell.

`USERNAME` and `PASSWORD` are for a user with access to the test app associated
with `CLIENT_ID`. The test app should not have 2FA enabled and allow password
login. The General Settings for the test app should have Application Grant type
with Implicit (hybrid) enabled.

```
go test -test.v
```
