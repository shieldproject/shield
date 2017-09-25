package store

import (
	"sort"
	"strconv"
)

type MemoryStore struct {
	db map[string]Configuration
}

var dbCounter int

func NewMemoryStore() MemoryStore {
	dbCounter = 0
	return MemoryStore{db: make(map[string]Configuration)}
}

func (store MemoryStore) Put(name string, value string) (string, error) {
	config := Configuration{
		Name:  name,
		Value: value,
		ID:    strconv.Itoa(dbCounter),
	}
	dbCounter++

	store.db[config.ID] = config
	return config.ID, nil
}

func (store MemoryStore) GetByName(name string) (Configurations, error) {
	var results Configurations

	for _, config := range store.db {
		if config.Name == name {
			results = append(results, config)
		}
	}

	sort.Sort(results)

	return results, nil
}

func (store MemoryStore) GetByID(id string) (Configuration, error) {
	return store.db[id], nil
}

func (store MemoryStore) Delete(name string) (int, error) {
	deletedCount := 0

	for _, config := range store.db {
		if config.Name == name {
			delete(store.db, config.ID)
			deletedCount++
		}
	}

	return deletedCount, nil
}
