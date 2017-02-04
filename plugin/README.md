# Shield Plugins

## Example Configurations

### Postgres
```
{
  "pg_user"     : "username-for-postgres",
  "pg_password" : "password-for-above-user",
  "pg_host"     : "hostname-or-ip-of-pg-server",
  "pg_port"     : "port-above-pg-server-listens-on", # optional
  "pg_database" : "name-of-db-to-backup",            # optional
  "pg_bindir"   : "PostgreSQL binaries directory"    # optional
}
```
Defaults:
```
{
  "pg_port"   : "5432",
  "pg_bindir" : "/var/vcap/packages/postgres/bin"
}
```

### MySQL
```
{
  "mysql_host"         : "127.0.0.1",    # optional
  "mysql_port"         : "3306",         # optional
  "mysql_user"         : "username",
  "mysql_password"     : "password",
  "mysql_read_replica" : "hostname/ip",  # optional
  "mysql_database"     : "db",           # optional
  "mysql_options"      : "--quick",      # optional
  "mysql_bindir"       : "/path/to/bin"  # optional
}
```
Defaults:
```
{
  "mysql_host"   : "127.0.0.1",
  "mysql_port"   : "3306",
  "mysql_bindir" : "/var/vcap/packages/shield-mysql/bin"
}
```

### Consul
```
{
  "host"     : "host-or-ip:port",           # optional
  "username" : "consul-http-auth-username", # optional
  "password" : "consul-http-auth-password"  # optional
```
Defaults:
```
{
  "host" : "127.0.0.1:8500"
}
```

### Mongo
```
{
  "mongo_host"     : "127.0.0.1",   # optional
  "mongo_port"     : "27017",       # optional
  "mongo_user"     : "username",    # optional
  "mongo_password" : "password",    # optional
  "mongo_database" : "db",          # optional
  "mongo_bindir"   : "/path/to/bin" # optional
}
```
Defaults:
```
{
  "mongo_host"     : "127.0.0.1",   # optional
  "mongo_port"     : "27017"        # optional
}
```

### Fill out more plugins in this readme and submit a PR
