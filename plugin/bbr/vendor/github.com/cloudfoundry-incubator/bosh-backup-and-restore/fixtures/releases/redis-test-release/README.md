# redis-example-service-release

A BOSH release for Redis.

This is an example intended to be deployed on demand by a [Cloud Foundry service broker](http://docs.pivotal.io/on-demand-service-broker).
One BOSH deployment of this release corresponds to one service instance.

There is an example manifest in the `development` directory. It must be used on
a version of BOSH that supports global cloud config (246 or higher).

**Please note that this release is meant for demonstration purposes only, not for production use.**
