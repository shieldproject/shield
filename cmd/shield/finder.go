package main

import (
	"fmt"
	"strings"

	"github.com/pborman/uuid"

	. "github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

func FindStore(s ...string) (Store, uuid.UUID, error) {
	search := strings.Join(s, " ")

	id := uuid.Parse(search)
	if id != nil {
		s, err := GetStore(id)
		if err == nil {
			return s, uuid.Parse(s.UUID), err
		}
		return s, nil, err
	}

	stores, err := GetStores(StoreFilter{
		Name: search,
	})
	if err != nil {
		return Store{}, nil, fmt.Errorf("Failed to retrieve list of archive stores from SHIELD: %s", err)
	}

	switch len(stores) {
	case 0:
		return Store{}, nil, fmt.Errorf("no matching archive stores found")

	case 1:
		return stores[0], uuid.Parse(stores[0].UUID), nil

	default:
		t := tui.NewTable("Name", "Summary", "Plugin", "Configuration")
		for _, store := range stores {
			t.Row(store, store.Name, store.Summary, store.Plugin, store.Endpoint)
		}
		want := tui.Menu(
			fmt.Sprintf("More than one archive store matched your search for '%s':", search),
			&t, "Which archive store do you wanh?")
		return want.(Store), uuid.Parse(want.(Store).UUID), nil
	}
}
