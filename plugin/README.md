# Shield Plugins

## Example Configurations

### Postgres
```
{
    "pg_user":"username-for-postgres",
    "pg_password":"password-for-above-user",
    "pg_host":"hostname-or-ip-of-pg-server",
    "pg_port":"port-above-pg-server-listens-on", # optional
    "pg_database": "name-of-db-to-backup",       # optional
    "pg_bindir": "PostgreSQL binaries directory" # optional
}
```
Defaults:
```
{
    "pg_port":"5432",
    "pg_bindir": "/var/vcap/packages/postgres/bin"
}
```

### MySQL
```
    {
        "mysql_user":"username-for-mysql",
        "mysql_password":"password-for-above-user",
        "mysql_host":"hostname-or-ip-of-mysql-server",
        "mysql_port":"port-above-mysql-server-listens-on",
        "mysql_read_replica":"hostname-or-ip-of-mysql-replica-server",  #OPTIONAL
        "mysql_database": "your-database-name",  #OPTIONAL
       	"mysql_options": "mysqldump-specific-options", #OPTIONAL
        "mysql_bindir": "/mysql/binary/directory" #OPTIONAL
    }
```
Defaults:
```
{
  "mysql_port": "3306",
  "mysql_bindir":  "/var/vcap/packages/shield-mysql/bin"
}
```

### Fill out more plugins in this readme and submit a PR
