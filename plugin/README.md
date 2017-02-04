# Shield Plugins

| Plugin                              | Config Name     | Target | Store |
| ----------------------------------- | ----------------| :----: | :---: |
| [Consul](#consul)                   | consul          | X      |       |
| [Docker Postgres](#docker-postgres) | docker-postgres | X      |       |
| [Filesystem](#filesystem)           | fs              | X      | X     |
| [MongoDB](#mongodo)                 | mongo           | X      |       |
| [Postgres](#postgres)               | postgres        | X      |       |
| [MySQL](#mysql)                     | mysql           | X      |       |
| [RabbitMQ Broker](#rabbmq-broker)   | rabbitmq-broker | X      |       |
| [Redis Broker](#redis-broker)       | redis-broker    | X      |       |
| [S3](#s3)                           | s3              |        | X     |
| [Scality](#scality)                 | scality         |        | X     |
| [Xtra Backup](#xtrabackup)          | xtrabackup      | X      |       |

**See a missing plugin? Create one and submit a PR**<br>
[Example Plugin Stub](../plugin/dummy)

## Consul
[Plugin Source](../plugin/consul/plugin.go)

| Property    | Value  |
| ----------- | ------ |
| Config Name | consul |
| Target      | yes    |
| Store       | no     |

**Configuration**
```
{
  // OPTIONAL
  "host"     : "host-or-ip:port",
  "username" : "consul-http-auth-username",
  "password" : "consul-http-auth-password"
}
```

**Defaults**
```
{
  "host" : "127.0.0.1:8500"
}
```

## Docker Postgres
[Plugin Source](../plugin/docker-postgres/plugin.go)

| Property    | Value           |
| ----------- | --------------- |
| Config Name | docker-postgres |
| Target      | yes             |
| Store       | no              |
**Configuration**
```
{}
```

**Defaults**
```
{}
```

## Filesystem
[Plugin Source](../plugin/fs/plugin.go)

| Property    | Value           |
| ----------- | --------------- |
| Config Name | fs              |
| Target      | yes             |
| Store       | no              |

**Configuration**
```
{
    "base_dir" : "base-directory-to-backup",
    // OPTIONAL
    "include" : "glob-of-files-to-include",
    "exclude" : "glob-of-files-to-exclude",
    "bsdtar"  : "/var/vcap/packages/bsdtar/bin/bsdtar"
}
```

**Defaults**
```
{
  "bsdtar": "/var/vcap/packages/bsdtar/bin/bsdtar"
}
```

## Mongo
[Plugin Source](../plugin/mongo/plugin.go)

| Property    | Value |
| ----------- | ----- |
| Config Name | mongo |
| Target      | yes   |
| Store       | no    |

**Configuration**
```
{
  // OPTIONAL
  "mongo_host"     : "127.0.0.1",
  "mongo_port"     : "27017",
  "mongo_user"     : "username",
  "mongo_password" : "password",
  "mongo_database" : "db",
  "mongo_bindir"   : "/path/to/bin"
}
```

**Defaults**
```
{
  "mongo_host"        : "127.0.0.1",
  "DefaultPort"       : "27017",
  "DefaultMongoBinDir": "/var/vcap/packages/shield-mongo/bin"
}
```

## Postgres
[Plugin Source](../plugin/postgres/plugin.go)

| Property    | Value    |
| ----------- | -------- |
| Config Name | postgres |
| Target      | yes      |
| Store       | no       |

**Configuration**
```
{
  "pg_user"    : "username-for-postgres",
  "pg_password": "password-for-above-user",
  "pg_host"    : "hostname-or-ip-of-pg-server",
  // OPTIONAL
  "pg_port"    : "port-above-pg-server-listens-on",
  "pg_database": "name-of-db-to-backup",
  "pg_bindir"  : "PostgreSQL binaries directory"
}
```

**Defaults**
```
{
  "pg_port"  : "5432",
  "pg_bindir": "/var/vcap/packages/postgres/bin"
}
```

## MySQL
[Plugin Source](../blob/master/plugin/mysql/plugin.go)

| Property    | Value |
| ----------- | ----- |
| Config Name | mysql |
| Target      | yes   |
| Store       | no    |

**Configuration**
```
{
  "mysql_user"         : "username",
  "mysql_password"     : "password",
  // OPTIONAL
  "mysql_host"         : "127.0.0.1",
  "mysql_port"         : "3306",
  "mysql_read_replica" : "hostname/ip",
  "mysql_database"     : "db",
  "mysql_options"      : "--quick",
  "mysql_bindir"       : "/path/to/bin"
}
```

**Defaults**
```
{
  "mysql_host"   : "127.0.0.1",
  "mysql_port"   : "3306",
  "mysql_bindir" : "/var/vcap/packages/shield-mysql/bin"
}
```

## RabbitMQ Broker
[Plugin Source](../plugin/rabbitmq-broker/plugin.go)

| Property    | Value           |
| ----------- | --------------- |
| Config Name | rabbitmq-broker |
| Target      | yes             |
| Store       | no              |

**Configuration**
```
{
  "rmq_url"             :"url-to-rabbitmq-management-domain",
  "rmq_username"        :"basic-auth-user-for-above-domain",
  "rmq_password"        :"basic-auth-passwd-for-above-domain",
  // OPTIONAL
  "skip_ssl_validation" :false
}
```

**Defaults**
```
{
  "skip_ssl_validation": false
}
```

## Redis Broker
[Plugin Source](../plugin/rabbitmq-broker/plugin.go)

| Property    | Value        |
| ----------- | ------------ |
| Config Name | redis-broker |
| Target      | yes          |
| Store       | no           |

**Configuration**
```
{
  "redis_type": "<dedicated|broker>"
}
```

**Defaults**
```
{}
```

## S3
[Plugin Source](../plugin/rabbitmq-broker/plugin.go)

| Property    | Value |
| ----------- | ----- |
| Config Name | s3    |
| Target      | no    |
| Store       | yes   |

**Configuration**
```
{
    "s3_host"             : "s3.amazonaws.com",
    "access_key_id"       : "your-access-key-id",
    "secret_access_key"   : "your-secret-access-key",
    "skip_ssl_validation" : false,
    "bucket"              : "bucket-name",
    "prefix"              : "/path/inside/bucket/to/place/backup/data",
    "signature_version"   : "4",
    // OPTIONAL
    "socks5_proxy": "",
    "s3_port"     : ""
}
```

**Defaults**
```
{
  "s3_host"             : "s3.amazonawd.com",
  "signature_version"   : "4",
  "skip_ssl_validation" : false
}
```

## Scality
[Plugin Source](../plugin/rabbitmq-broker/plugin.go)

| Property    | Value   |
| ----------- | ------- |
| Config Name | scality |
| Target      | no      |
| Store       | yes     |

**Configuration**
```
{
  "scality_host"        : "your-scality-host",
  "access_key_id"       : "your-access-key-id",
  "secret_access_key"   : "your-secret-access-key",
  "bucket"              : "bucket-name",
  // OPTIONAL
  "skip_ssl_validation" : false,
  "prefix"              : "/path/inside/bucket/to/place/backup/data",
  "socks5_proxy"        : ""
}
```

**Defaults**
```
{
  "skip_ssl_validation" : false,
  "prefix"              : "",
  "socks5_proxy"        : ""
}
```

## Xtra Backup
[Plugin Source](../plugin/rabbitmq-broker/plugin.go)

| Property    | Value      |
| ----------- | ---------- |
| Config Name | xtrabackup |
| Target      | yes        |
| Store       | no         |

**Configuration**
```
{
  "mysql_user"          : "username-for-mysql",
  "mysql_password"      : "password-for-above-user",
  // OPTIONAL
  "mysql_databases"     : <list_of_databases>,
  "mysql_datadir"       : "/var/lib/mysql",
  "mysql_xtrabackup"    : "/path/to/xtrabackup",
  "mysql_temp_targetdir": "/tmp/backups"
  "mysql_tar"           : "tar"
}
```

**Defaults**
```
{
  "mysql_tar"           : "tar",
  "mysql_datadir"       : "/var/lib/mysql",
  "mysql_xtrabackup"    : "/var/vcap/packages/shield-mysql/bin/xtrabackup",
  "mysql_temp_targetdir": "/tmp/backups"
}
```
