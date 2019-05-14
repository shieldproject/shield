Getting Started
===============

Hi!

Welcome to the SHIELD Getting Started guide.  This document is for
you if you;

  1. Care about your data systems
  2. Want to run SHIELD to protect your data
  3. Have access to Docker

That's it!

After you've run through this guide, you'll have a working SHIELD
installation, with a configured data system, a successful backup
job, an archive, and you'll have run through a restore.  More
importantly, we hope you'll be comfortable working with SHIELD,
and ready to explore more of what you can do with it.



Deploying SHIELD
----------------

We're going to use [Docker Compose][1] to spin up SHIELD and all
of its components (there's quite a few of them).

    $ mkdir ~/shield-demo
    $ cd ~/shield-demo
    $ curl -sLO $raw/docker-compose.yml

Now that you have the compose recipe file locally, let's fire it
up.

    $ docker-compose up

SHIELD is now running at https://localhost:9009.  If you visit it
in your browser of choice, you should see something like this:

![The SHIELD Login Page]($docs/ops/getting-started/login.png)

The default credentials for SHIELD are `admin` (username) and
`shield` (password).  After you log in, you will be presented with
the _initialization screen_, where you'll set your master
password, to protect the encryption parameters of all backup
archives.

![The SHIELD Initialization Page]($docs/ops/getting-started/init.png)

After you've set your master password (via the form on the right),
SHIELD generates a _fixed key_ which isn't terribly important
right now but is very important in production.  It's used to
encrypt the backup archives of the SHIELD metadata, for disaster
recovery.

![The Fixed Key]($docs/ops/getting-started/fixed-key.png)

Copy that fixed key into your password manager for safe-keeping
and click the "I Understand" button to continue.

You should now be all logged into SHIELD.

![SHIELD Interface]($docs/ops/getting-started/home.png)



Your First Backup Job
---------------------




Restoring the Data
------------------


Backing up SHIELD Itself
------------------------
In a disaster recover situations is important to get shield back up and
running as soon as possible to facilitate the restore of the your other
systems.

    "Name": "SHIELD"
    "Notes": "SHIELD Backup"
    "Agent": "alex-lab-shield/shield@z1/0"
    "Backup Plugin": "Local Filesystem (fs)"
    "Base Directory": "/var/vcap/store/shield"
    "Files to Include": <blank>*
    "Files to Exclude": <blank>*
    "Fixed-Key Encryption?": <checked>
    "Strict Mode": <unchecked>


[1]: https://docs.docker.com/compose/overview/
