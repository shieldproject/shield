package store

type IRows interface {
	Next() bool
	Close() error
	Scan(dest ...interface{}) error
}
