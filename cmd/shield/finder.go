package main

import (
	"fmt"
	"time"

	"github.com/pborman/uuid"

	. "github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

func FindStore(search string, strict bool) (Store, uuid.UUID, error) {
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
		return Store{}, nil, fmt.Errorf("Failed to retrieve list of archive stores: %s", err)
	}

	switch len(stores) {
	case 0:
		return Store{}, nil, fmt.Errorf("no matching archive stores found")

	case 1:
		return stores[0], uuid.Parse(stores[0].UUID), nil

	default:
		if strict {
			return Store{}, nil, fmt.Errorf("more than one matching archive store found")
		}
		t := tui.NewTable("Name", "Summary", "Plugin", "Configuration")
		for _, store := range stores {
			t.Row(store, store.Name, store.Summary, store.Plugin, store.Endpoint)
		}
		want := tui.Menu(
			fmt.Sprintf("More than one archive store matched your search for '%s':", search),
			&t, "Which archive store do you want?")
		return want.(Store), uuid.Parse(want.(Store).UUID), nil
	}
}

func FindTarget(search string, strict bool) (Target, uuid.UUID, error) {
	id := uuid.Parse(search)
	if id != nil {
		want, err := GetTarget(id)
		if err != nil {
			return Target{}, nil, err
		}
		return want, uuid.Parse(want.UUID), nil
	}

	targets, err := GetTargets(TargetFilter{
		Name: search,
	})
	if err != nil {
		return Target{}, nil, fmt.Errorf("Failed to retrieve list of backup targets: %s", err)
	}
	switch len(targets) {
	case 0:
		return Target{}, nil, fmt.Errorf("no matching backup targets found")

	case 1:
		return targets[0], uuid.Parse(targets[0].UUID), nil

	default:
		if strict {
			return Target{}, nil, fmt.Errorf("more than one matching backup target found")
		}
		t := tui.NewTable("Name", "Summary", "Plugin", "Configuration")
		for _, target := range targets {
			t.Row(target, target.Name, target.Summary, target.Plugin, target.Endpoint)
		}
		want := tui.Menu(
			fmt.Sprintf("More than one backup target matched your search for '%s':", search),
			&t, "Which backup target do you want?")
		return want.(Target), uuid.Parse(want.(Target).UUID), nil
	}
}

func FindRetentionPolicy(search string, strict bool) (RetentionPolicy, uuid.UUID, error) {
	id := uuid.Parse(search)
	if id != nil {
		want, err := GetRetentionPolicy(id)
		if err != nil {
			return RetentionPolicy{}, nil, err
		}
		return want, uuid.Parse(want.UUID), nil
	}

	policies, err := GetRetentionPolicies(RetentionPolicyFilter{
		Name: search,
	})
	if err != nil {
		return RetentionPolicy{}, nil, fmt.Errorf("Failed to retrieve list of retention policies: %s", err)
	}
	switch len(policies) {
	case 0:
		return RetentionPolicy{}, nil, fmt.Errorf("no matching retention policies found")

	case 1:
		return policies[0], uuid.Parse(policies[0].UUID), nil

	default:
		if strict {
			return RetentionPolicy{}, nil, fmt.Errorf("more than one matching retention policies found")
		}
		t := tui.NewTable("Name", "Summary", "Expires in")
		for _, policy := range policies {
			t.Row(policy, policy.Name, policy.Summary, fmt.Sprintf("%d days", policy.Expires/86400))
		}
		want := tui.Menu(
			fmt.Sprintf("More than one retention policy matched your search for '%s':", search),
			&t, "Which retention policy do you want?")
		return want.(RetentionPolicy), uuid.Parse(want.(RetentionPolicy).UUID), nil
	}
}

func FindSchedule(search string, strict bool) (Schedule, uuid.UUID, error) {
	id := uuid.Parse(search)
	if id != nil {
		want, err := GetSchedule(id)
		if err != nil {
			return Schedule{}, nil, err
		}
		return want, uuid.Parse(want.UUID), nil
	}

	schedules, err := GetSchedules(ScheduleFilter{
		Name: search,
	})
	if err != nil {
		return Schedule{}, nil, fmt.Errorf("Failed to retrieve list of backup schedules: %s", err)
	}
	switch len(schedules) {
	case 0:
		return Schedule{}, nil, fmt.Errorf("no matching backup schedules found")

	case 1:
		return schedules[0], uuid.Parse(schedules[0].UUID), nil

	default:
		if strict {
			return Schedule{}, nil, fmt.Errorf("more than one matching backup schedule found")
		}
		t := tui.NewTable("Name", "Summary", "Frequency / Interval (UTC)")
		for _, schedule := range schedules {
			t.Row(schedule, schedule.Name, schedule.Summary, schedule.When)
		}
		want := tui.Menu(
			fmt.Sprintf("More than one backup schedule matched your search for '%s':", search),
			&t, "Which backup schedule do you want?")
		return want.(Schedule), uuid.Parse(want.(Schedule).UUID), nil
	}
}

func FindJob(search string, strict bool) (Job, uuid.UUID, error) {
	id := uuid.Parse(search)
	if id != nil {
		want, err := GetJob(id)
		if err != nil {
			return Job{}, nil, err
		}
		return want, uuid.Parse(want.UUID), nil
	}

	jobs, err := GetJobs(JobFilter{
		Name: search,
	})
	if err != nil {
		return Job{}, nil, fmt.Errorf("Failed to retrieve list of jobs: %s", err)
	}
	switch len(jobs) {
	case 0:
		return Job{}, nil, fmt.Errorf("no matching jobs found")

	case 1:
		return jobs[0], uuid.Parse(jobs[0].UUID), nil

	default:
		if strict {
			return Job{}, nil, fmt.Errorf("more than one matching job found")
		}
		t := tui.NewTable("Name", "Summary", "Target", "Store", "Schedule")
		for _, job := range jobs {
			t.Row(job, job.Name, job.Summary, job.TargetName, job.StoreName, job.ScheduleWhen)
		}
		want := tui.Menu(
			fmt.Sprintf("More than one job matched your search for '%s':", search),
			&t, "Which backup job do you want?")
		return want.(Job), uuid.Parse(want.(Job).UUID), nil
	}
}

func FindArchivesFor(target Target, show int) (Archive, uuid.UUID, error) {
	archives, err := GetArchives(ArchiveFilter{
		Target: target.UUID,
		Status: "valid",
	})
	if err != nil {
		return Archive{}, nil, err
	}
	if len(archives) == 0 {
		return Archive{}, nil, fmt.Errorf("no valid backup archives found for target %s", target.Name)
	}

	if show > len(archives) {
		show = len(archives)
	} else {
		archives = archives[:show]
	}

	t := tui.NewTable("UUID", "Taken at", "Expires at", "Status", "Notes")
	for _, archive := range archives {
		t.Row(archive, archive.UUID,
			archive.TakenAt.Format(time.RFC1123Z),
			archive.ExpiresAt.Format(time.RFC1123Z),
			archive.Status, archive.Notes)
	}

	want := tui.Menu(
		fmt.Sprintf("Here are the %d most recent backup archives for target %s:", show, target.Name),
		&t, "Which backup archive would you like to restore?")

	return want.(Archive), uuid.Parse(want.(Archive).UUID), nil
}
