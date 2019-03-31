package core

import (
	"fmt"

	"github.com/starkandwayne/shield/db"
)

type Bearing struct {
	Tenant   *db.Tenant    `json:"tenant"`
	Archives []*db.Archive `json:"archives"`
	Jobs     []*db.Job     `json:"jobs"`
	Targets  []*db.Target  `json:"targets"`
	Stores   []*db.Store   `json:"stores"`
	Agents   []*db.Agent   `json:"agents"`
	Role     string        `json:"role"`

	Grants struct {
		Admin    bool `json:"admin"`
		Engineer bool `json:"engineer"`
		Operator bool `json:"operator"`
	} `json:"grants"`
}

func (c *Core) BearingFor(m *db.Membership) (Bearing, error) {
	var (
		b   Bearing
		err error
	)

	b.Tenant, err = c.db.GetTenant(m.TenantUUID)
	if err != nil {
		return b, fmt.Errorf("unable to retrieve tenant [%s]: %s", m.TenantUUID, err)
	}
	b.Role = m.Role
	switch b.Role {
	case "admin":
		b.Grants.Admin = true
		fallthrough
	case "engineer":
		b.Grants.Engineer = true
		fallthrough
	case "operator":
		b.Grants.Operator = true
	}

	b.Archives, err = c.db.GetAllArchives(&db.ArchiveFilter{ForTenant: b.Tenant.UUID})
	if err != nil {
		return b, fmt.Errorf("unable to retreieve archives for tenant [%s]: %s", b.Tenant.UUID, err)
	}

	/* assemble jobs for this tenant */
	b.Jobs, err = c.db.GetAllJobs(&db.JobFilter{ForTenant: b.Tenant.UUID})
	if err != nil {
		return b, fmt.Errorf("unable to retreive jobs for tenant [%s]: %s", b.Tenant.UUID, err)
	}

	/* assemble targets for this tenant */
	b.Targets, err = c.db.GetAllTargets(&db.TargetFilter{ForTenant: b.Tenant.UUID})
	if err != nil {
		return b, fmt.Errorf("unable to retrieve targets for tenant [%s]: %s", b.Tenant.UUID, err)
	}

	/* assemble stores for this tenant */
	b.Stores, err = c.db.GetAllStores(&db.StoreFilter{ForTenant: b.Tenant.UUID})
	if err != nil {
		return b, fmt.Errorf("unable to retrieve stores for tenant [%s]: %s", b.Tenant.UUID, err)
	}

	/* assemble agents and plugins for this tenant */
	b.Agents, err = c.db.GetAllAgents(&db.AgentFilter{SkipHidden: true, InflateMetadata: true})
	if err != nil {
		return b, fmt.Errorf("unable to retrieve agents for tenant [%s]: %s", b.Tenant.UUID, err)
	}

	return b, nil
}
