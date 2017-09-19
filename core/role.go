package core

import (
	"fmt"
	"os"
)

type Role string

const (
	SysAdminRole    Role = "sys_admin"
	SysManagerRole  Role = "sys_manager"
	SysEngineerRole Role = "sys_engineer"

	TenantAdminRole    Role = "tenant_admin"
	TenantEngineerRole Role = "tenant_engineer"
	TenantOperatorRole Role = "tenant_operator"
)

func (r Role) Can(right string) bool {
	if r == SysAdminRole {
		return true
	}

	switch right {
	/* manage_tenants

	   Governs the creation, editing, and deletion of Tenants
	*/
	case "manage_tenants":
		return r == SysManagerRole

	/* grant

	   Governs the assignment of local users to the Tenant,
	   and assignment of roles to Local Users

	   (Provider-based accounts MUST be granted roles per
	    their fixed auth provider configurations, in the
	    SHIELD daemon configuration)
	*/
	case "grant":
		return r == SysManagerRole || r == TenantAdminRole

	/* configure_global_storage

	   Governs the creation, editing, and deletion of globally
	   available Storage Endpoints, which are shared by Tenants
	   in a readonly fashion.
	*/
	case "configure_global_storage":
		return r == SysManagerRole || r == SysEngineerRole

	/* configure_retention_template

	   Governs the creation, editing, and deletion of globally
	   defined Retention Policy Templates, which are copied into
	   each new Tenant upon creation (i.e. NOT shared)
	*/
	case "configure_retention_template":
		return r == SysManagerRole || r == SysEngineerRole

	/* configure

	   Governs the creation, editing, and deletion of per-Tenant
	   storage endpoints, backup targets, and retention policies.
	   Roles without this right will NOT be able to view the
	   configuration of those entities, as they may contain sensitive
	   credentials.
	*/
	case "configure":
		return r == TenantAdminRole || r == TenantEngineerRole

	/* adhoc_run

	   Governs the ability to issue one-off, ad hoc runs of
	   defined backup jobs.
	*/
	case "adhoc_run":
		return r == TenantAdminRole || r == TenantEngineerRole || r == TenantOperatorRole

	/* pause_unpause

	   Governs the ability to pause or unpause defined jobs.
	*/
	case "pause_unpause":
		return r == TenantAdminRole || r == TenantEngineerRole || r == TenantOperatorRole

	/* restore

	   Governs the ability to initiate a restore of a
	   backup archive to a pre-defined target.
	*/
	case "restore":
		return r == TenantAdminRole || r == TenantEngineerRole || r == TenantOperatorRole

	/* adhoc_purge

	   Governs the ability to delete backup archives before
	   their expiry has passed.
	*/
	case "adhoc_purge":
		return r == TenantAdminRole || r == TenantEngineerRole || r == TenantOperatorRole
	}

	fmt.Fprintf(os.Stderr, "WARNING! Invalid right '%s' given to role.Can(); this is a bug.\n", right)
	return false
}
