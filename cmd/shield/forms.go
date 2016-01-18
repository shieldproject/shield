package main

import (
	"fmt"
	"strconv"

	"github.com/pborman/uuid"

	. "github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

func FieldIsStoreUUID(name string, value string) (interface{}, error) {
	id := uuid.Parse(value)
	if id != nil {
		want, err := GetStore(id)
		if err != nil {
			return nil, err
		}
		return want.UUID, nil
	}

	stores, err := GetStores(StoreFilter{
		Name: value,
	})
	if err != nil {
		return value, fmt.Errorf("Failed to retrieve list of archive stores from SHIELD: %s", err)
	}
	switch len(stores) {
	case 0:
		return value, fmt.Errorf("no matching archive stores found")
	case 1:
		return stores[0].UUID, nil
	default:
		t := tui.NewTable("Name", "Summary", "Plugin", "Configuration")
		for _, store := range stores {
			t.Row(store, store.Name, store.Summary, store.Plugin, store.Endpoint)
		}
		want := tui.Menu(
			fmt.Sprintf("More than one archive store matched your search query of '%s':", value),
			&t, "Which archive store do you want to use for this backup job?")
		return want.(Store).UUID, nil
	}
}

func FieldIsTargetUUID(name string, value string) (interface{}, error) {
	id := uuid.Parse(value)
	if id != nil {
		want, err := GetTarget(id)
		if err != nil {
			return nil, err
		}
		return want.UUID, nil
	}

	targets, err := GetTargets(TargetFilter{
		Name: value,
	})
	if err != nil {
		return value, fmt.Errorf("Failed to retrieve list of backup targets from SHIELD: %s", err)
	}
	switch len(targets) {
	case 0:
		return value, fmt.Errorf("no matching backup targets found")
	case 1:
		return targets[0].UUID, nil
	default:
		t := tui.NewTable("Name", "Summary", "Plugin", "Configuration")
		for _, target := range targets {
			t.Row(target, target.Name, target.Summary, target.Plugin, target.Endpoint)
		}
		want := tui.Menu(
			fmt.Sprintf("More than one backup target matched your search query of '%s':", value),
			&t, "Which backup target do you want to use for this backup job?")
		return want.(Target).UUID, nil
	}
}

func FieldIsRetentionPolicyUUID(name string, value string) (interface{}, error) {
	id := uuid.Parse(value)
	if id != nil {
		want, err := GetRetentionPolicy(id)
		if err != nil {
			return nil, err
		}
		return want.UUID, nil
	}

	policies, err := GetRetentionPolicies(RetentionPolicyFilter{
		Name: value,
	})
	if err != nil {
		return value, fmt.Errorf("Failed to retrieve list of retention policies from SHIELD: %s", err)
	}
	switch len(policies) {
	case 0:
		return value, fmt.Errorf("no matching retention policies found")
	case 1:
		return policies[0].UUID, nil
	default:
		t := tui.NewTable("Name", "Summary", "Expires in")
		for _, policy := range policies {
			t.Row(policy, policy.Name, policy.Summary, fmt.Sprintf("%d days", policy.Expires/86400))
		}
		want := tui.Menu(
			fmt.Sprintf("More than one retention policy matched your search query of '%s':", value),
			&t, "Which retention policy do you want to use for this backup job?")
		return want.(RetentionPolicy).UUID, nil
	}
}

func FieldIsScheduleUUID(name string, value string) (interface{}, error) {
	id := uuid.Parse(value)
	if id != nil {
		want, err := GetSchedule(id)
		if err != nil {
			return nil, err
		}
		return want.UUID, nil
	}

	schedules, err := GetSchedules(ScheduleFilter{
		Name: value,
	})
	if err != nil {
		return value, fmt.Errorf("Failed to retrieve list of backup schedules from SHIELD: %s", err)
	}
	switch len(schedules) {
	case 0:
		return value, fmt.Errorf("no matching backup schedules found")
	default:
		t := tui.NewTable("Name", "Summary", "Frequency / Interval (UTC)")
		for _, schedule := range schedules {
			t.Row(schedule, schedule.Name, schedule.Summary, schedule.When)
		}
		want := tui.Menu(
			fmt.Sprintf("More than one backup schedule matched your search query of '%s':", value),
			&t, "Which backup schedule do you want to use for this backup job?")
		return want.(Schedule).UUID, nil
	}
}

func FieldIsRetentionTimeframe(name string, value string) (interface{}, error) {
	i, err := strconv.Atoi(value)
	if err != nil {
		return value, fmt.Errorf("'%s' is not an integer: %s", value, err)
	}
	if i < 0 {
		return value, fmt.Errorf("retention timeframe must be at least 1 day")
	}
	return i * 86400, nil
}
