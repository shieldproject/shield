package main

import (
	//"encoding/json"
	"fmt"
	"os"

	"github.com/pborman/uuid"

	. "github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

type ListRetentionOptions struct {
	Unused bool
	Used   bool
	UUID   string
}

func ListRetentionPolicies(opts ListRetentionOptions) error {
	policies, err := GetRetentionPolicies(RetentionPoliciesFilter{
		Unused: MaybeBools(opts.Unused, opts.Used),
	})
	if err != nil {
		return fmt.Errorf("failed to retrieve retention policies from SHIELD: %s", err)
	}
	t := tui.NewTable("UUID", "Name", "Description", "Expires in")
	for _, policy := range policies {
		if len(opts.UUID) > 0 && opts.UUID == policy.UUID {
			t.Row(policy.UUID, policy.Name, policy.Summary, fmt.Sprintf("%d days", policy.Expires/86400))
			break
		} else if len(opts.UUID) > 0 && opts.UUID != policy.UUID {
			continue
		}
		t.Row(policy.UUID, policy.Name, policy.Summary, fmt.Sprintf("%d days", policy.Expires/86400))
	}
	t.Output(os.Stdout)
	return nil
}

func CreateNewRetentionPolicy() error {
	content := invokeEditor(`{
		"name":     "Empty Retention Policy",
		"summary":  "Should probably tell me how long I should keep this",
		"expires":  86400
		}`)
	newPolicy, err := CreateRetentionPolicy(content)
	fmt.Printf("The new policy is: %v\n", newPolicy)
	if err != nil {
		return fmt.Errorf("ERROR: Could not create new retention policy: %s", err)
	}
	fmt.Fprintf(os.Stdout, "Created new retention policy.\n")
	t := tui.NewTable("UUID", "Name", "Description", "Expires in")
	t.Row(newPolicy.UUID, newPolicy.Name, newPolicy.Summary, fmt.Sprintf("%d days", newPolicy.Expires/86400))
	t.Output(os.Stdout)
	return nil
}

func EditExstingPolicy(u string) error {
	p, err := GetRetentionPolicy(uuid.Parse(u))
	if err != nil {
		return fmt.Errorf("ERROR: Cannot retrieve policy '%s': %s", u, err)
	}
	content := invokeEditor(`{
		"name":     "` + p.Name + `",
		"summary":  "` + p.Summary + `",
		"expires":  ` + fmt.Sprintf("%d", p.Expires) + `
		}`)
	p, err = UpdateRetentionPolicy(uuid.Parse(u), content)
	if err != nil {
		return fmt.Errorf("ERROR: Cannot update policy '%s': %s", u, err)
	}
	fmt.Fprintf(os.Stdout, "Updated policy.\n")
	t := tui.NewTable("UUID", "Name", "Description", "Expires in")
	t.Row(p.UUID, p.Name, p.Summary, fmt.Sprintf("%d days", p.Expires/86400))
	t.Output(os.Stdout)
	return nil
}

func DeleteRetentionPolicyByUUID(u string) error {
	err := DeleteRetentionPolicy(uuid.Parse(u))
	if err != nil {
		return fmt.Errorf("ERROR: Cannot delete retention policy '%s': %s", u, err)
	}
	fmt.Fprintf(os.Stdout, "Deleted retention policy '%s'\n", u)
	return nil
}
