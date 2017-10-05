# Shield Docker Compose
Made for learning and trying out [Shield](https://github.com/starkandwayne/shield) locally

- [Setup](#setup)
- [WebUI](#webui)
- [CLI](#cli)
- [Tutorial](#tutorial)
  - [Terminology](#terminology)
  - [Backends](#backends)
  - [Targets](#targets)
  - [Stores](#stores)
  - [Policies](#policies)
  - [Jobs](#jobs)
    - [Manually Run Backup](#manually-run-backup)
  - [Restore](#restore-from-backup)

## Setup
Install [docker-compose](https://docs.docker.com/compose/install/)
```
git clone git@github.com:starkandwayne/shield.git 
cd shield-docker
docker-compose build
```

### Start up
```
docker-compose up
```

**If you make any changes you will most likely need to rebuild.**
```
docker-compose build
```

### Teardown
```
docker-compose down
```

### WebUI
Check out the WebUI in your browser of choice
```
https://<docker-host>
```

If you are using the latest docker setup `<docker-host>` should be `localhost`

The system is using a self-signed cert so your browser will most likely complain.

Shields Credentials:
```
username: user
password: password
```

### CLI
Install shield CLI
```
brew tap starkandwayne/cf
brew install starkandwayne/cf/shield
```
Download CLI from https://github.com/starkandwayne/shield/releases

---
## Tutorial
### Terminology
- Backend - shield daemon server
- Target - service to backup
- Store - place to store backups
- Policy - how long to keep backups (ie 10 days, 30 days, etc)
- Job - Runs a backup/restore

The shield daemon serves up a `backend` api

We `target` a service to `backup/restore` to/from a `store` which has a `policy` to set how long to `store` a `backup`

`Backups` can be `scheduled` with a `job`

Start up shield:
```
docker-compose up
```

### Sample Data
The included `postgres` database has a table called `people` and is populated with a few rows.

| id  | name  |
| --- | ----- |
| 1   | Bob   |
| 2   | Sarah |
| 3   | Tim   |

We will use this to make/restore backups


### Backends
In order to use a backend lets create one!
```
shield create-backend lh https://localhost
```

Check that everything works:
```
shield status -k
```

response:
```
Using https://localhost (lh) as SHIELD backend

Authentication Required

User: user

Password: password

Name:
API Version: X.X.X
```

the `-k` is to skip ssl validation since we are using a self-sign cert for docker-compose

The response should look something like:
```
Using https://localhost (lh) as SHIELD backend

Name:
API Version: <version>
```

You can have multiple `backends` and can list them with:

```
shield backends
```

You can pick a backend to use by:

```
shield backend <backend-name>
```

### Targets
**Services to Backup/Restore**

**Create a target**
```
shield create-target -k
```

You will then be prompted to ask several questions:

```
Using https://localhost (lh) as SHIELD backend

Target Name:    my-target-name
Summary:        optional summary
Plugin Name:    postgres
Configuration:  { "pg_user": "postgres", "pg_password": "postgres", "pg_host": "postgres", "pg_database": "shield", "pg_bindir": "/usr/lib/postgresql/9.6/bin" }
Remote IP:port: shield-agent:4222


Really create this target? [y/n] y
```

Response:

```
Created new target
Name:          my-target-name
Summary:       optional summary

Plugin:        postgres
Configuration: { "pg_user": "postgres", "pg_password": "postgres", "pg_host": "postgres", "pg_database": "shield", "pg_bindir": "/usr/lib/postgresql/9.6/bin" }
Remote IP:     shield-agent:4222
```

List our targets:
```
shield targets -k
```

```
Using https://localhost (lh) as SHIELD backend

Name            Summary           Plugin    Remote IP               Configuration
====            =======           ======    =========               =============
my-target-name  optional summary  postgres  shield-agent:4222  {
                                                                      "pg_user": "postgres",
                                                                      "pg_password": "postgres",
                                                                      "pg_host": "postgres",
                                                                      "pg_database": "shield",
                                                                      "pg_bindir": "/usr/lib/postgresql/9.6/bin"
                                                                    }
```

### Stores
**Where to store our backups**

Create a Store
```
shield create-store -k
```

Fill out some fields
```
Using https://localhost (lh) as SHIELD backend

Store Name: filesystem
Summary: some summary
Plugin Name: fs
Configuration (JSON): {"base_dir": "/tmp"}


Store Name:           filesystem
Summary:              some summary
Plugin Name:          fs
Configuration (JSON): {"base_dir": "/tmp"}


Really create this archive store? [y/n] y
```

Response:
```
Created new store
Name:          filesystem
Summary:       some summary

Plugin:        fs
Configuration: {"base_dir": "/tmp"}
```

List our stores:
```
shield stores -k
```

```
Using https://localhost (lh) as SHIELD backend

Name        Summary       Plugin  Configuration
====        =======       ======  =============
filesystem  some summary  fs      {
                                    "base_dir": "/tmp"
                                  }
```

### Policies
**Pick how long backups will be stored**

Create a policy

```
shield create-policy -k
```


Fill in additional fields
```
Using https://localhost (lh) as SHIELD backend

Policy Name: 10-day
Summary: optional summary
Retention Timeframe, in days: 10


Policy Name:                  10-day
Summary:                      optional summary
Retention Timeframe, in days: 10

Really create this retention policy? [y/n] y
```

Response
```
Created new retention policy
Name:       10-day
Summary:    optional summary
Expiration: 10 days
```

List policies
```
shield policies -k
```

Response:
```
Using https://localhost (lh) as SHIELD backend

Name    Summary           Expires in
====    =======           ==========
10-day  optional summary  10 days
```

### Jobs
Create a job
```
shield create-job -k
```

Fill in fields
```
Using https://localhost (lh) as SHIELD backend

Job Name: my-job
Summary: optional summary
Store: filesystem
Target: my-target-name
Retention Policy: 10-day
Schedule: daily 4am
Paused? (no): no


Job Name:         my-job
Summary:          optional summary
Store:            filesystem (af4fd251-3754-425f-b83c-f5597c40043b)
Target:           my-target-name (5e177a44-36a2-4683-ab9c-150494bc43ab)
Retention Policy: 10-day (d642a1f6-7f88-4ff5-aba0-308fb61866bb)
Schedule:         daily 4am
Paused?:          false


Really create this backup job? [y/n] y
```

Response:
```
Created new job
Name:             job
Paused:           N

Retention Policy: 10-day
Expires in:       10 days

Schedule:         daily 4am

Target:           postgres
Target Endpoint:  { "pg_user": "postgres", "pg_password": "postgres", "pg_host": "postgres", "pg_database": "shield", "pg_bindir": "/usr/lib/postgresql/9.6/bin" }
Remote IP:        shield-agent:4222

Store:            fs
Store Endpoint:   {"base_dir": "/tmp"}

Notes:            optional summary
```

### Manually Run Backup
```
shield run my-job -k
```

This will create an instance of a job called a task and execute it

response:
```
Using https://localhost (lh) as SHIELD backend

Scheduled immediate run of job
To view task, type shield task <task-uuid>
```

Get detailed information about our task. The <task-uuid> will be unique so look at the response from the previous command to get your <task-uuid>
```
shield task <task-uuid> -k
```

This will print out a detailed list of everything running. This is useful to see what is going on under the hood.

### Restore from Backup
```
shield restore -k
```

This will put you into an interactive prompt where you can pick a backup that you want to use with your restore

```
Using https://localhost (lh) as SHIELD backend

Here are the 1 most recent backup archives for target my-target-name:

      UUID                                  Taken at                         Expires at                       Status  Notes
      ====                                  ========                         ==========                       ======  =====
   1) 9c512425-63ee-446b-8077-a8875ccb3a19  Tue, 31 Jan 2017 20:48:10 +0000  Fri, 10 Feb 2017 20:48:10 +0000  valid


  Which backup archive would you like to restore? [1-1]
```



## Additional Commands
```
shield commands
```

### TODO
- preload example schedule/target/backup/job...?
- create docker shield image on docker hub
- create docker agent image on docker hub
- run CLI from a container
