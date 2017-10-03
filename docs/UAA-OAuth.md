# UAA

UAA  can be used as an external OAuth2 provider for controlling SHIELD authorizations.

This will work for both standalone UAA instances, and UAA instances part of a Cloud Foundry. This means existing authorization structures can be carried over into SHIELD with minimal additional setup;  Define a tenant name for SHIELD to display in its menus and CLI, and SHIELD will give users the proper authorization based on their SCIM rights.

## Have a working UAA instance

The UAA boshrelease can be found [here](https://github.com/cloudfoundry/uaa-release), with the UAA docs being located [here](https://docs.cloudfoundry.org/uaa/)

## Install cf-uaac Gem

```shell
gem install cf-uaac
uaac version
```

## Authenticate to UAA

```shell
uaac target --skip-ssl-validation http(s)://<the uaa address>

uaac token client get admin

<enter the admin client password>
#note that this differs from the admin user password
```

## Create a new UAA Client for SHIELD

```shell
uaac client add shield-dev \
  --name S.H.I.E.L.D. \
  --scope openid \
  --authorities uaa.none \
  --authorized_grant_types authorization_code \
  --redirect_uri http://<YOUR_SHIELD_ADDRESSS_HERE>/auth/oauth/<YOUR_UAA_IDENTIFIER_FROM_YOUR_SHIELD_CONFIG_HERE>/redir \
  --access_token_validity  180 \
  --refresh_token_validity 180 \
  --secret s.h.i.e.l.d.
```

SHIELD requires the `openid` scope as this scope is required to access the /userinfo endpoint, which gives SHIELD the various standard user profile/group fields that it needs in order to exectue on the mapping in the configuration.

## Add the configuration for the UAA OAuth client

This will be added to the already existing configuration YAML file used to deploy shield under a new field called `auth`

### required fields are:

- __name__ - Displayed in the UI and CLI when interacting with SHIELD
- __identifier__ -  The SHIELD backend will use this to identify which OAuth Client was in use for things like logs and auditing.
- __backend__ - A static value defined by SHIELD; It will always be `uaa` for this configuration
- __properties__:
  - __client_id__ - Decided by the admin during UAA client creation.
  - __client_secret__ - Decided by the admin during UAA client creation.
  - __mapping__ -  Define a name for the UAA SHIELD tenant, and define a map from a given user's SCIM rights to predefined SHIELD roles.
   SHIELD can also accept a base case role for all users that do not belong to one of the matching SCIM rights provided.
    - __tenant__ - Any name.  SHIELD will create an internal SHIELD tenant with this given name
    - __rights__ - Maps a user's SCIM rights to a predefined SHIELD role. Can include a default role for all users that don't possess any of the listed SCIM rights.
      - __scim__ - One of the predefined UAA SCIM rights, which can be found [here](https://github.com/cloudfoundry/uaa/blob/master/docs/UAA-APIs.rst#scopes-authorized-by-the-uaa)
      - __role__ - One of three predefined SHIELD roles
        - admin
        - operator
        - engineer

### UAA section of config used for SHIELD development:

```yaml
auth:
  - name:       UAA
    identifier: uaa1
    backend:    uaa
    properties:
      client_id:       shield-dev
      client_secret:   s.h.i.e.l.d.
      uaa_endpoint:    https://uaa.shield.10.244.156.2.netip.cc:8443
      skip_verify_tls: true
      mapping:
        - tenant: UAA          # <-- shield tenant name
          rights:
            - scim: uaa.admin  # <-- uaa scim right
              role: admin      # <-- shield role
                               #   (first match wins)
            - scim: cloud_controller.write
              role: engineer

            - role: operator   # = (default match)

        - tenant: UAA Admins Club
          rights:
            - scim: uaa.admin
              role: admin
```
