# Changelog

## v2.1.1 (June 27th, 2025)

### Updates:

* Update jwx library [#120](https://github.com/okta/okta-jwt-verifier-golang/pull/129)
* Update CI configuration [#121](https://github.com/okta/okta-jwt-verifier-golang/pull/130)

## v2.1.0 (September 5th, 2024)

### Updates:

Make Audience validation optional [#116](https://github.com/okta/okta-jwt-verifier-golang/pull/116)

## v2.0.5 (September 3rd, 2024)

### Updates:

Make audience optional [#117](https://github.com/okta/okta-jwt-verifier-golang/pull/117)
Update jwx library [#114](https://github.com/okta/okta-jwt-verifier-golang/pull/114)

## v2.0.4 (February 26th, 2024)

### Updates:

Update staticcheck [#110](https://github.com/okta/okta-jwt-verifier-golang/pull/110)

## v2.0.3 (July 28th, 2023)

### Updates:

* Fixing the race condition [#102](https://github.com/okta/okta-jwt-verifier-golang/pull/102)

## v2.0.2 (May 18th, 2023)

### Updates:

* Correct okta-jwt-verifier-golang version reference to v2  [#101](https://github.com/okta/okta-jwt-verifier-golang/pull/101)

## v2.0.1 (May 15th, 2023)

### Enhancements:

* Customizable HTTP client [#99](https://github.com/okta/okta-jwt-verifier-golang/pull/99)

### Updates:

* Project maintenance for CI [#95](https://github.com/okta/okta-jwt-verifier-golang/pull/95)
* Correct logging typo [#91](https://github.com/okta/okta-jwt-verifier-golang/pull/91)
* Replace `math/rand` with `crypto/rand` [#89](https://github.com/okta/okta-jwt-verifier-golang/pull/89)

## v2.0.0 (January 4th, 2023)

### Enhancements:

* Customizable cache timeout and change to the cache method. [#92](https://github.com/okta/okta-jwt-verifier-golang/pull/92)

## v1.3.1 (April 6th, 2022)

### Updates:

* Correctly error if metadata from issuer is not 200. [#85](https://github.com/okta/okta-jwt-verifier-golang/pull/85). Thanks, [@monde](https://github.com/monde)!

## v1.3.0 (March 17th, 2022)

### Enhancements:

* New PCKE code verifier utility. [#81](https://github.com/okta/okta-jwt-verifier-golang/pull/81). Thanks, [@deepu105](https://github.com/deepu105)!

## v1.2.1 (February 16, 2022)

### Updates

* Update JWX package. Thanks, [@thomassampson](https://github.com/thomassampson)!

## v1.2.0 (February 16, 2022)

### Updates

* Customizable resource cache. Thanks, [@tschaub](https://github.com/tschaub)!

## v1.1.3

### Updates

- Fixed edge cause with `aud` claim that would not find Auth0 being JWTs valid. Thanks [@awrenn](https://github.com/awrenn)!
- Updated readme with testing notes.
- Ran `gofumpt` on code for clean up.

## v1.1.2

### Updates

- Only `alg` and `kid` claims in a JWT header are considered during verification.
