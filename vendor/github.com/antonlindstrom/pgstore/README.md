# pgstore

A session store backend for [gorilla/sessions](http://www.gorillatoolkit.org/pkg/sessions) - [src](https://github.com/gorilla/sessions).

## Installation

    make get-deps

## Documentation

Available on [godoc.org](http://www.godoc.org/github.com/antonlindstrom/pgstore).

See http://www.gorillatoolkit.org/pkg/sessions for full documentation on underlying interface.

### Example

```go
// Fetch new store.
store := NewPGStore("postgres://user:password@127.0.0.1:5432/database?sslmode=verify-full", []byte("secret-key"))
defer store.Close()
// Run a background goroutine to clean up expired sessions from the database.
defer store.StopCleanup(store.Cleanup(time.Minute * 5))

// Get a session.
session, err = store.Get(req, "session-key")
if err != nil {
    log.Error(err.Error())
}

// Add a value.
session.Values["foo"] = "bar"

// Save.
if err = sessions.Save(req, rsp); err != nil {
    t.Fatalf("Error saving session: %v", err)
}

// Delete session.
session.Options.MaxAge = -1
if err = sessions.Save(req, rsp); err != nil {
    t.Fatalf("Error saving session: %v", err)
}
```

## Thanks

I've stolen, borrowed and gotten inspiration from the other backends available:

* [redistore](https://github.com/boj/redistore)
* [mysqlstore](https://github.com/srinathgs/mysqlstore)
* [babou dbstore](https://github.com/drbawb/babou/blob/master/lib/session/dbstore.go)

Thank you all for sharing your code!

What makes this backend different is that it's for Postgresql and uses the fine
datamapper [Gorp](https://github.com/coopernurse/gorp).
Make sure you use a somewhat new codebase of Gorp as it now defaults to text for
strings when it used to default to varchar 255. Varchar 255 is unfortunately too
small.
