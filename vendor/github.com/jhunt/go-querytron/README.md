Querytron
=========

![Travis CI](https://travis-ci.org/jhunt/go-querytron.svg?branch=master)

Following hot on the successes of [Envirotron][env], **Querytron**
is here to save the day!

Ever needed to deal with a remote system that dealt in both
`application/json` _AND_ `application/x-www-form-urlencoded` data?

No?  I see you've never had the misfortune of integrating with
OAuth2 providers!  Good on you then.

For the rest of us, I wrote Querytron.  It works a lot like
Envirotron:

```
package thing

import (
  "fmt"
  qs "github.com/jhunt/go-querytron"
)

type Response struct {
  Error string `qs:"error"`
  URI   string `qs:"error_uri"`
}

func main() {
  url := SomeFunction()
  var r Response
  qs.Override(&r, url.Query())

  fmt.Printf("error %s (see also %s)\n", c.Error, c.URI)
}
```

Querytron also works in the other direction, generating
querystrings for you, from structure definitions.  Given a
structure like:

```
type Example struct {
  Query string `qs:"q"`
  Limit *int   `qs:"limit"`
  Fuzzy *bool  `qs:"fuzzy:yes"`
}
```

A call to `qs.Generate(&Example{...})` will generate a
`url.Values` object and return it, such that:

- **q=...** is set if `Query` is anything besides the empty string
- **limit=...** is set if `Limit` is a non-nil int pointer
- **fuzzy=y** is set if `Fuzzy` is non-nil and points to true

This should make query-string based API client interfaces easier
to write.

Oh, and if you (like me) are a bit miffed that Go doesn't let you
take the address of literals, there's a whole suite of
address-taking functions, like `qs.Int(...)`, `qs.Uint64()`, etc.,
as well as two pointer-booleans, `qs.True` and `qs.False`.

Happy Hacking!

[env]: https://github.com/jhunt/go-envirotron
