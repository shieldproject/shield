sqlitestore
==========

Gorilla's Session Store Implementation for SQLite

Installation
===========

Run `go get github.com/michaeljs1990/sqlitestore` from command line. Gets installed in `$GOPATH`

Usage
=====

`NewSqliteStore` takes the following paramaters

    endpoint - A sql.Open style endpoint
    tableName - table where sessions are to be saved. Required fields are created automatically if the table doesnot exist.
    path - path for Set-Cookie header
    maxAge
    codecs

Internally, `sqlitestore` uses [this](https://github.com/mattn/go-sqlite3) SQLite driver.

e.g.,


      package main

      import (
  	    "fmt"
  	    "github.com/michaeljs1990/sqlitestore"
  	    "net/http"
      )

      var store, _ = sqlitestore.NewSqliteStore("./database", "sessions", "/", 3600, []byte("<SecretKey>"))
      defer store.Close()

      func sessTest(w http.ResponseWriter, r *http.Request) {
  	    session, err := store.Get(r, "foobar")
  	    session.Values["bar"] = "baz"
  	    session.Values["baz"] = "foo"
  	    err = session.Save(r, w)
  	    fmt.Printf("%#v\n", session)
  	    fmt.Println(err)
      }

    func main() {
    	http.HandleFunc("/", sessTest)
    	http.ListenAndServe(":8080", nil)
    }
