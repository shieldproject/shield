# Building new docker image

```
vagrant up --provider=virtualbox
vagrant ssh
```

On vagrant VM:

```
cd /opt/bosh-agent/ci/docker
docker login ...
sudo ./build_docker_image.sh
```
