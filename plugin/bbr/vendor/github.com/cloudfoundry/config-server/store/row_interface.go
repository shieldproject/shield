package store

type IRow interface {
	Scan(dest ...interface{}) error
}
