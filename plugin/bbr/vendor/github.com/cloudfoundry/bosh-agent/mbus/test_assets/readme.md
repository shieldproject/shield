The certs in this directory will expire on 15 May 2018. Regenerate them by then using:

```
# /tmp/manifest.yml

---
variables:
- name: default_ca
  type: certificate
  options:
    is_ca: true
- name: my_keys
  type: certificate
  options:
    common_name: localhost
    ca: default_ca
```

extract and save the values like

```
$ bosh int --vars-store creds.yml /tmp/manifest.yml
$ bosh int creds.yml --path /my_keys/certificate > custom_cert.pem
$ bosh int creds.yml --path /my_keys/private_key > custom_key.pem
$ bosh int creds.yml --path /my_keys/ca > custom_ca.pem
```