package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

//List available retention policies
func cliListPolicies(args ...string) error {
	DEBUG("running 'list retention policies' command")
	DEBUG("  show unused? %v", *opts.Unused)
	DEBUG("  show in-use? %v", *opts.Used)
	if *opts.Raw {
		DEBUG(" fuzzy search? %v", api.MaybeBools(*opts.Fuzzy, *opts.Raw).Yes)
	}

	policies, err := api.GetRetentionPolicies(api.RetentionPolicyFilter{
		Name:       strings.Join(args, " "),
		Unused:     api.MaybeBools(*opts.Unused, *opts.Used),
		ExactMatch: api.Opposite(api.MaybeBools(*opts.Fuzzy, *opts.Raw)),
	})
	if err != nil {
		return err
	}

	if *opts.Raw {
		return RawJSON(policies)
	}

	t := tui.NewTable("Name", "Summary", "Expires in")
	for _, policy := range policies {
		t.Row(policy, policy.Name, policy.Summary, fmt.Sprintf("%d days", policy.Expires/86400))
	}
	t.Output(os.Stdout)
	return nil
}

func cliGetPolicy(args ...string) error {
	DEBUG("running 'show retention policy' command")

	policy, _, err := FindRetentionPolicy(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if *opts.Raw {
		return RawJSON(policy)
	}
	if *opts.ShowUUID {
		return RawUUID(policy.UUID)
	}

	ShowRetentionPolicy(policy)
	return nil
}

func cliCreatePolicy(args ...string) error {
	DEBUG("running 'create retention policy' command")

	var err error
	var content string
	if *opts.Raw {
		content, err = readall(os.Stdin)
		if err != nil {
			return err
		}

	} else {
		in := tui.NewForm()
		in.NewField("Policy Name", "name", "", "", tui.FieldIsRequired)
		in.NewField("Summary", "summary", "", "", tui.FieldIsOptional)
		in.NewField("Retention Timeframe, in days", "expires", "", "", FieldIsRetentionTimeframe)

		if err := in.Show(); err != nil {
			return err
		}

		if !in.Confirm("Really create this retention policy?") {
			return errCanceled
		}

		content, err = in.BuildContent()
		if err != nil {
			return err
		}
	}

	DEBUG("JSON:\n  %s\n", content)

	if *opts.UpdateIfExists {
		t, id, err := FindRetentionPolicy(content, true)
		if err != nil {
			return err
		}
		if id != nil {
			t, err = api.UpdateRetentionPolicy(id, content)
			if err != nil {
				return err
			}
			MSG("Updated existing retention policy")
			return cliGetPolicy(t.UUID)
		}
	}

	p, err := api.CreateRetentionPolicy(content)

	if err != nil {
		return err
	}

	MSG("Created new retention policy")
	return cliGetPolicy(p.UUID)
}

//Modify an existing retention policy
func cliEditPolicy(args ...string) error {
	DEBUG("running 'edit retention policy' command")

	p, id, err := FindRetentionPolicy(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	var content string
	if *opts.Raw {
		content, err = readall(os.Stdin)
		if err != nil {
			return err
		}

	} else {
		in := tui.NewForm()
		in.NewField("Policy Name", "name", p.Name, "", tui.FieldIsRequired)
		in.NewField("Summary", "summary", p.Summary, "", tui.FieldIsOptional)
		in.NewField("Retention Timeframe, in days", "expires", p.Expires/86400, "", FieldIsRetentionTimeframe)

		if err = in.Show(); err != nil {
			return err
		}

		if !in.Confirm("Save these changes?") {
			return errCanceled
		}

		content, err = in.BuildContent()
		if err != nil {
			return err
		}
	}

	DEBUG("JSON:\n  %s\n", content)
	p, err = api.UpdateRetentionPolicy(id, content)
	if err != nil {
		return err
	}

	MSG("Updated retention policy")
	return cliGetPolicy(p.UUID)
}

//Delete a retention policy
func cliDeletePolicy(args ...string) error {
	DEBUG("running 'delete retention policy' command")

	policy, id, err := FindRetentionPolicy(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if !*opts.Raw {
		ShowRetentionPolicy(policy)
		if !tui.Confirm("Really delete this retention policy?") {
			return errCanceled
		}
	}

	if err := api.DeleteRetentionPolicy(id); err != nil {
		return err
	}

	OK("Deleted policy")
	return nil
}
