package main

import (
	//"encoding/json"
	"fmt"
	"os"

	"github.com/pborman/uuid"

	. "github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

type ListStoreOptions struct {
	Unused bool
	Used   bool
	Plugin string
	UUID   string
}

func ListStores(opts ListStoreOptions) error {
	//FIXME: (un)?used flags not working; --plugin works.
	stores, err := GetStores(StoreFilter{
		Plugin: opts.Plugin,
		Unused: MaybeBools(opts.Unused, opts.Used),
	})
	if err != nil {
		return fmt.Errorf("ERROR: Could not fetch list of stores: %s", err)
	}
	t := tui.NewTable("UUID", "Name", "Description", "Plugin", "Endpoint")
	for _, store := range stores {
		//FIXME: use GetStore(UUID)
		if len(opts.UUID) > 0 && opts.UUID == store.UUID {
			t.Row(store.UUID, store.Name, store.Summary, store.Plugin, store.Endpoint)
			break
		} else if len(opts.UUID) > 0 && opts.UUID != store.UUID {
			continue
		}
		t.Row(store.UUID, store.Name, store.Summary, store.Plugin, store.Endpoint)
	}
	t.Output(os.Stdout)
	return nil
}

func CreateNewStore() error {
	content := invokeEditor(`{
		"name":     "Empty Store",
		"summary":  "It would be fun to open my own shop",
		"plugin":   "seriously",
		"endpoint": "{\"items\":\"unknown\"}"
		}`)
	newStore, err := CreateStore(content)
	if err != nil {
		return fmt.Errorf("ERROR: Could not create new store: %s", err)
	}
	fmt.Fprintf(os.Stdout, "Created new store.\n")
	t := tui.NewTable("UUID", "Name", "Description", "Plugin", "Endpoint")
	t.Row(newStore.UUID, newStore.Name, newStore.Summary, newStore.Plugin, newStore.Endpoint)
	t.Output(os.Stdout)
	return nil
}

func EditExstingStore(u string) error {
	s, err := GetStore(uuid.Parse(u))
	if err != nil {
		return fmt.Errorf("ERROR: Cannot retrieve store '%s': %s", u, err)
	}

	content := invokeEditor(`{
		"name":     "` + s.Name + `",
		"summary":  "` + s.Summary + `",
		"plugin":   "` + s.Plugin + `",
		"endpoint": "` + s.Endpoint + `"
		}`)

	s, err = UpdateStore(uuid.Parse(u), content)
	if err != nil {
		return fmt.Errorf("ERROR: Cannot update store '%s': %s", u, err)
	}
	fmt.Fprintf(os.Stdout, "Updated store.\n")
	t := tui.NewTable("UUID", "Name", "Description", "Plugin", "Endpoint")
	t.Row(s.UUID, s.Name, s.Summary, s.Plugin, s.Endpoint)
	t.Output(os.Stdout)
	return nil
}

func DeleteStoreByUUID(u string) error {
	err := DeleteStore(uuid.Parse(u))
	if err != nil {
		return fmt.Errorf("ERROR: Cannot delete store '%s': %s", u, err)
	}
	fmt.Fprintf(os.Stdout, "Deleted store '%s'\n", u)
	return nil
}
