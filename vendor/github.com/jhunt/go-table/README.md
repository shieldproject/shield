go-table
========

![Travis CI](https://travis-ci.org/jhunt/go-table.svg?branch=master)

A small library for formatting fluid tables for the CLI.

Usage
-----

You start with a `table.NewTable()`, to define your headers:

```go
t := table.NewTable("#", "Name", "Notes", "Status")
```

Then, for each thing you want to tabularize, just call `t.Row()`,
passing it a nil first argument and the values for each column:

```go
t.Row(nil, 1, "Foo", "lorem ipsum dolor sit amet...", "GOOD")
t.Row(nil, 2, "Bar", "you can even have\nembedded newlines...", "GOOD")
```

Finally, to print it, call `t.Output()`, passing it the io.Writer
you want it to print to (like `os.Stdout`):

```go
t.Output(os.Stdout)
```

Those four lines render the following table, effortlessly:

```
#  Name  Notes                          Status
=  ====  =====                          ======
1  Foo   lorem ipsum dolor sit amet...  GOOD
2  Bar   you can even have              GOOD
         embedded newlines...
```
