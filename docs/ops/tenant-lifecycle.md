Tenant Lifecycle
================


Tenant Deletion
---------------

If you wish to delete a tenant the shield cli provides you with the `delete-tenant <tenant-name>`
command that will allow you to delete the tenant provided there is no existing configuration
under that tenant.
Configuration meaning any outstanding:
  1. tasks
  2. memberships
  3. jobs
  4. targets
  5. stores
  6. archives

If you wish to delete a tenant and recursively delete the above configuration use the
`delete-tenant -r <tenant-name>`command.

*Note after confirming the deletion, the above will be deleted from shield and archives will be purged from storage.*