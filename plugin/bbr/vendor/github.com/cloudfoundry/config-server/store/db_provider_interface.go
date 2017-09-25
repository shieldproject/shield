package store

type DbProvider interface {
	Db() (IDb, error)
}
