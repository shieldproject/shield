# Okta Authentication

Okta can be used as an external OAuth2 provider for controlling
SHIELD authentication and authorizations.  Both public Okta
(okta.com) and private Enterprise Okta are supported.

All you as a SHIELD site operator need to do is configure the
SHIELD Core with a new authentication provider backend, and
register an OAuth Application via the Okta web interface.

## Registering a Okta OAuth Application

To start, access the _Applications_ page by expanding the Applications on the left panel menu of the Okta web interface:

Then, click on the _Create App Integration_ button to bring up
the form for registering your SHIELD to Okta:

Then, just fill out the form:

For _Sign-in method_, select `OIDC - OpenID Connect`.
_Application type_ will be `Web Application`.
<!-- FIXME: need better screenshots.
            less emphasis on navigation,
            more emphasis on the form to fill out -->

The `Sign-in redirect URIs` field must be set to:

    https://$shield/auth/$identifier/redir

Where `https://$shield` is the address of the SHIELD instance, and
`$identifier` is the name you configured the authentication provider
with, via the SHIELD Core configuration file.

Okta will generate a new Client ID and Client Secret.  Take note
of these, as they are necessary for the next step.

After this, we will need a _groups claim_ set up to access user-mapping between SHIELD users and Okta users.

For that, navigate to _Security -> API_. Take a note of the `authorization server` and click on the one you need to use. 

Add a `groups scope` under _scope tabs_ and a `groups claim` under _claims tab_ with the value being `"groups: matches regex .*"`.

## Configuring SHIELD

To configure SHIELD to work with Okta, you need to add a new
_authentication provider_ configuration stanza to the SHIELD Core
configuration file:

    # ... all the other shield core configuration ...

    auth:
      - identifier: okta  # or whatever you used when registering
        name:       Okta
        backend:    okta
        properties:
          client_id:            YOUR-OKTA-CLIENT-ID
          client_secret:        YOUR-OKTA-CLIENT-SECRET
          okta_domain:          YOUR-OKTA-DOMAIN       
          authorization_server: YOUR-OKTA-AUTH-SERVER
          deployment_uri:       SHIELD-DEPLOYMENT-URL
          token_verification:   true/false                #OPTIONAL

          mapping:  []    # more on this later

The `auth` key is a list of all configured authentication
providers; if your configuration already features other providers,
like token or UAA, you will just need to append the Okta
configuration to that.

The top-level of each `auth` item has the following required keys:

  - **identifier** - An internal name, used by SHIELD to
    differentiate this authentication provider configuration from all
    of the others.  This is used in the Okta Application
    Redirect URL, so it should not be changed lightly.

  - **name** - A human-friendly name that will be displayed to
    web and CLI users when they are trying to decide which
    authentication method they wish to use.

  - **backend** - What provider backend to use.  For Okta, this
    will always be `okta`.

  - **properties** - Properties specific to the Okta
    authentication provider.  Detailed next.

### Configuring Okta Authentication Properties

The `properties` key has the following sub-keys:

  - **client\_id** - The Okta Client ID for your registered
    OAuth application, available from the Okta web interface.

  - **client\_secret** - The Okta Client Secret for your
    registered OAuth application, available from the Okta web
    interface.

  - **okta\_domain** -  The organization URL for your Okta account. 

  - **authorization\_server** -  The Authorization Server for your Okta account, found under Security -> API.

  - **deployment\uri** -  The address at which your SHIELD is deployed. This is used to construct the redirect URI for OKTA OAuth redirect handling. 

  - **token_verification** -  OPTIONAL - This value tells SHIELD to add further validation of the access and id token it receives from Okta after authentication. 

  - **mapping** - A list of rules for mapping Okta organizations
    and groups to SHIELD tenants and roles.

### Mappings

Each element of the `properties.mapping` list specifies a rule for
translating Okta's organizations and groups into SHIELD tenants
and roles.  This rule scheme allows for a great deal of
flexibility in bridging the two systems, allowing you to mix
multiple Okta orgs into a single SHIELD tenant, split a single
org into multiple tenants, etc.

The format of each rule is:

    - okta: Okta Organization Name
      tenant: SHIELD Tenant Name
      rights:
        - group: Okta Group Name
          role: SHIELD Role
        # ... etc ...

The `okta` field matches the Okta organization name.  If a
user is found to belong to this organization, the rest of the rule
is processed.  Processing starts by looking through the list of
`rights`, until a match on `group` to Okta group name is found, at
which point the specified `role` is assigned to the user, on the
given `tenant`.

Here's an example:

    auth:
      - identifier: okta
        name:       Okta
        backend:    okta
        properties:
          client_id:            YOUR-OKTA-CLIENT-ID
          client_secret:        YOUR-OKTA-CLIENT-SECRET
          okta_domain:          YOUR-OKTA-DOMAIN       
          authorization_server: YOUR-OKTA-AUTH-SERVER
          deployment_uri:       SHIELD-DEPLOYMENT-URL
          token_verification:   true/false                #OPTIONAL

          mapping: 
            - okta: okta-org
              tenant: okta-tenant
              rights:
                - group: Admin
                  role: admin
                - group: Users
                  role: engineer
                - role: operator

In this configuration, SHIELD will assign someone in the _Owners_
group of the _okta-org_ org to the _okta-tenant_ SHIELD tenant, as an _admin_.  Members of the
_Users_ Okta group (on the same org) who are not in _Owners_
will be assigned the _engineer_ role instead.  Everyone else in
the Okta org will be assigned the _operator_ role.

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
rights and roles to Okta users, based on the same rules as
tenant-level role assignment.

The _SYSTEM_ tenant has its own set of assignable roles:

- **admin** - Full control over all of SHIELD.
- **manager** - Control over tenants and manual role assigments.
- **engineer** - Control over shared resources like global storage
  definitions and retention policy templates.
