# Github Authentication

Github can be used as an external OAuth2 provider for controlling
SHIELD authentication and authorizations.  Both public Github
(github.com) and private Enterprise Github are supported.

All you as a SHIELD site operator need to do is configure the
SHIELD Core with a new authentication provider backend, and
register an OAuth Application via the Github web interface.

## Registering a Github OAuth Application

To start, access the _Settings_ page by expanding the account
fly-out menu underneath your profile picture, in the top
right-hand side of the Github web interface:

![Accessing Settings : Screenshot](github2.png)

Then, access the _Oauth Apps_ panel, under the _Developer
Settings_ header on the left side of the screen:

![GitHub Setup](github3.png)

Then, click on the _Register a new application_ button to bring up
the form for registering your SHIELD to Github:

![GitHub Setup](github4.png)

Then, just fill out the form:

<!-- FIXME: need better screenshots.
            less emphasis on navigation,
            more emphasis on the form to fill out -->

The `Authorization callback URL` field must be set to:

    https://$shield/auth/$identifier/redir

Where `https://$shield` is the address of the SHIELD instance, and
`$identifier` is the name you configured the authentication provider
with, via the SHIELD Core configuration file.

Github will generate a new Client ID and Client Secret.  Take note
of these, as they are necessary for the next step.

## Configuring SHIELD

To configure SHIELD to work with Github, you need to add a new
_authentication provider_ configuration stanza to the SHIELD Core
configuration file:

    # ... all the other shield core configuration ...

    auth:
      - identifier: github  # or whatever you used when registering
        name:       Github
        backend:    github
        properties:
          client_id:      YOUR-GITHUB-CLIENT-ID
          client_secret:  YOUR-GITHUB-CLIENT-SECRET

          mapping:  []    # more on this later

The `auth` key is a list of all configured authentication
providers; if your configuration already features other providers,
like token or UAA, you will just need to append the Github
configuration to that.

The top-level of each `auth` item has the following required keys:

  - **identifier** - An internal name, used by SHIELD to
    differentiate this authentication provider configuration from all
    of the others.  This is used in the Github Application
    Redirect URL, so it should not be changed lightly.

  - **name** - A human-friendly name that will be displayed to
    web and CLI users when they are trying to decide which
    authentication method they wish to use.

  - **backend** - What provider backend to use.  For Github, this
    will always be `github`.

  - **properties** - Properties specific to the Github
    authentication provider.  Detailed next.

### Configuring Github Authentication Properties

The `properties` key has the following sub-keys:

  - **client\_id** - The Github Client ID for your registered
    OAuth application, available from the Github web interface.

  - **client\_secret** - The Github Client Secret for your
    registered OAuth application, available from the Github web
    interface.

  - **github\_endpoint** - (not shown above) An optional URL where
    SHIELD can find the Github endpoint.  This is primarily used
    for Enterprise Github customers who run an isolated,
    on-premise Github.  Public Github users need not set this.

  - **mapping** - A list of rules for mapping Github organizations
    and teams to SHIELD tenants and roles.

### Mappings

Each element of the `properties.mapping` list specifies a rule for
translating Github's organizations and teams into SHIELD tenants
and roles.  This rule scheme allows for a great deal of
flexibility in bridging the two systems, allowing you to mix
multiple Github orgs into a single SHIELD tenant, split a single
org into multiple tenants, etc.

The format of each rule is:

    - github: Github Organization Name
      tenant: SHIELD Tenant Name
      rights:
        - team: Github Team Name
          role: SHIELD Role
        # ... etc ...

The `github` field matches the Github organization name.  If a
user is found to belong to this organization, the rest of the rule
is processed.  Processing starts by looking through the list of
`rights`, until a match on `team` to Github team name is found, at
which point the specified `role` is assigned to the user, on the
given `tenant`.

Here's an example:

    auth:
      - identifier: github
        name:       Github
        backend:    github
        properties:
          client_id:      YOUR-GITHUB-CLIENT-ID
          client_secret:  YOUR-GITHUB-CLIENT-SECRET

          mapping:
            - github: cloudfoundry-community
              tenant: Cloud Foundry Community
              rights:
                - team: Owners
                  role: admin
                - team: Engineers
                  role: engineer
                - role: operator

In this configuration, SHIELD will assign someone in the _Owners_
team of the _cloudfoundry-community_ org to the _Cloud Foundry
Community_ SHIELD tenant, as an _admin_.  Members of the
_Engineers_ Github team (on the same org) who are not in _Owners_
will be assigned the _engineer_ role instead.  Everyone else in
the Github org will be assigned the _operator_ role.

These `rights` rules are processed until one matches; subsequent
rules are skipped.

If there is more than one mapping, each of them is tried, in
order;  this can lead to multiple tenant assignments for a single
user, which provides a lot of power to the SHIELD site operator.

Tenants that do not already exist in the database, but have been
defined in the authentication configuration, will be created as
needed.

Valid values for the `role` field are:

- **admin** - Full control over the tenant
- **engineer** - Control over the configuration of stores,
  targets, retention policies, and jobs.
- **operator** - Control over running jobs, pausing and unpausing
  scheduled jobs, and performing restore operations.

### The SYSTEM Tenant

There is a special tenant, called the _SYSTEM_ tenant, that exists
solely to allow SHIELD site operators to assign system-level
rights and roles to Github users, based on the same rules as
tenant-level role assignment.

The _SYSTEM_ tenant has its own set of assignable roles:

- **admin** - Full control over all of SHIELD.
- **manager** - Control over tenants and manual role assigments.
- **engineer** - Control over shared resources like global storage
  definitions and retention policy templates.
