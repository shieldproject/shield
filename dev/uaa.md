# UAA Testing

This document details the steps necessary to set up UAA for use in testing
and developing SHIELD and the UAA Authentication Provider.  This setup is
specifically geared towards use with `./bin/testdev`

This configuration gives you a single admin user with the password "PASSWORD"

## Deploy UAA

Deploy the uaa-release with the manifest in the SHIELD repo, at `dev/uaa.yml`

```
bosh -e env upload release ...
bosh -e env deploy -d shield-test-uaa dev/uaa.yml
```

## Install cf-uaac Gem

```
gem install cf-uaac
uaac version
```

## Authenticate to UAA

```
uaac target --skip-ssl-validation https://uaa.shield.10.244.156.2.netip.cc:8443

uaac token client get admin
```

(the password is `adminsecret`)

## Create a new UAA Client for SHIELD

```
uaac client add shield-dev \
  --name S.H.I.E.L.D. \
  --scope openid \
  --authorities uaa.none \
  --authorized_grant_types authorization_code \
  --redirect_uri http://localhost:8181/auth/oauth/uaa1/redir \
  --access_token_validity  180 \
  --refresh_token_validity 180 \
  --secret s.h.i.e.l.d.
```
