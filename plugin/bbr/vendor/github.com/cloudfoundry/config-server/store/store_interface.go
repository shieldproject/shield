package store

type Store interface {
	Put(key string, value string) (string, error)
	GetByName(name string) (Configurations, error)
	GetByID(id string) (Configuration, error)
	Delete(key string) (int, error)
}
