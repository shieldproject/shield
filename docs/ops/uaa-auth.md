# UAA Authentication

[Cloud Foundry UAA][1] can be used as an external OAuth2 provider
for controlling SHIELD authentication and authorizations.  BOSH
UAA, Cloud Foundry UAA, and standalone UAA instances can all be
used with this authentication provider.

All you as a SHIELD site operator need to do is configure the
SHIELD Core with a new authentication provider backend, and
register an Oauth Application via the `uaac` CLI for UAA.

## If You Need a UAA Instance...

We have provided a [deployment manifest][2] for the UAA BOSH
release, which is available [on Github][3]. Documentation for the
UAA itself can be found [here][4].

[2]: /dev/uaa.yml
[3]: https://github.com/cloudfoundry/uaa-release
[4]: https://docs.cloudfoundry.org/uaa

You will also need to install `uaac`, the UAA CLI utility, which
is packaged as a Ruby gem:

    gem install cf-uaac
    uaac version

## Registering A Client with UAA

First, you will need to authenticate to UAA:

    uaac target --skip-ssl-validation http(s)://<the uaa address>

    uaac token client get admin

    <enter the admin client password>
    #note that this differs from the admin user password

Then, create a new client for SHIELD to use:

    uaac client add $CLIENT_ID \
      --name $CLIENT_NAME \
      --scope openid \
      --authorities uaa.none \
      --authorized_grant_types authorization_code \
      --redirect_uri https://$SHIELD/auth/$IDENTIFIER/redir \
      --access_token_validity  180 \
      --refresh_token_validity 180 \
      --secret $CLIENT_SECRET

Where:

- **$CLIENT\_ID** is a unique identifier for the SHIELD UAA
  Client.  "shield" is a good value to use for this.

- **$CLIENT\_SECRET** is a secret, randomly generated password.

- **$SHIELD** is the hostname/IP and (optionally) port that SHIELD
  is reachable at.

- **$IDENTIFIER** is the authentication provider identifier you
  are going to use inside of the SHIELD Core configuraiton.  "uaa"
  is a good value.

SHIELD requires the `openid` scope as this scope is required to
access the `/userinfo` endpoint, which gives SHIELD the various
standard user profile/group fields that it needs in order to
map SCIM rights to SHIELD tenants and roles.

## Configuring SHIELD

To configure SHIELD to work with your UAA, you need to add a new
_authentication provider_ configuration stanza to the SHIELD Core
configuration file:

    # ... all the other shield core configuration ...

    auth:
      - identifier: uaa     # or whatever you used when registering
        name:       Cloud Foundry UAA
        backend:    uaa
        properties:
          client_id:      YOUR-UAA-CLIENT-ID
          client_secret:  YOUR-UAA-CLIENT-SECRET
          uaa_endpoint:   https://uaa.shield.10.244.156.2.netip.cc:8443

          mapping:  []    # more on this later

The `auth` key is a list of all configured authentication
providers; if your configuration already features other providers,
like token or Github, you will just need to append the UAA
configuration to that.

The top-level of each `auth` item has the following required keys:

  - **identifier** - An internal name, used by SHIELD to
    differentiate this authentication provider configuration from all
    of the others.  This is used in the UAA Application Redirect
    URL, so it should not be changed lightly.

  - **name** - A human-friendly name that will be displayed to
    web and CLI users when they are trying to decide which
    authentication method they wish to use.

  - **backend** - What provider backend to use.  For UAA, this
    will always be `uaa`.

  - **properties** - Properties specific to the UAA
    authentication provider.  Detailed next.

### Configuring UAA Authentication Properties

The `properties` key has the following sub-keys:

  - **client\_id** - The UAA Client ID for your registered
    OAuth application, which you provided when you registered the
    client via `uaac`.

  - **client\_secret** - The UAA Client Secret for your
    registered OAuth application, which you provided when you
    registered the client via `uaac`

  - **uaa\_endpoint** - The URL of your UAA instance.

  - **mapping** - A list of rules for mapping UAA SCIM rights to
    SHIELD tenants and roles.

### Mappings

Each element of the `properties.mapping` list specifies a rule for
translating UAA SCIM rights into SHIELD tenants and roles.  This
rule scheme allows for a great deal of flexibility in bridging the
two systems.

The format of each rule is:

    tenant: SHIELD Tenant Name
    rights:
      - scim: scim.right.name
        role: SHIELD Role
      # ... etc ...

Rules are processed by first looking through the list of `rights`,
until a `scim` right is found that the authenticated user has been
granted (by UAA).  Then, that user is granted access to the named
`tenant`, with the identified `role`, and the next rule is tried.

Here's an example:

    auth:
      - identifier: uaa
        name:       Cloud Foundry UAA
        backend:    uaa
        properties:
          client_id:      YOUR-UAA-CLIENT-ID
          client_secret:  YOUR-UAA-CLIENT-SECRET
          uaa_endpoint:   https://uaa.shield.10.244.156.2.netip.cc:8443

          mapping:
            - tenant: Stark & Wayne
              rights:
                - scim: uaa.admin
                  role: admin
                - scim: uaa.write
                  role: engineer
                - role: operator

In this configuration, SHIELD will assign someone with the
`uaa.admin` SCIM right to the _Stark & Wayne_ SHIELD tenant, as an
_admin_.  Anyone with the `uaa.write` SCIM right who don't have
`uaa.admin` will be assigned the _engineer_ role instead.
Everyone else who can authenticate to this UAA will be assigned
the _operator_ role.

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
