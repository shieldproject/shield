.PHONY: all build certs clean down ps

define create_folder_structure
	@mkdir -p core/shield/data
	@mkdir -p vault/vault/storage
	@mkdir -p mariadb/db
endef

all:
ifeq (,$(wildcard ./core/shield/id_rsa))
	$(MAKE) certs
endif
	$(call create_folder_structure)
	docker-compose up

build:
ifeq (,$(wildcard ./core/shield/id_rsa))
	$(MAKE) certs
endif
	$(call create_folder_structure)
	docker-compose build

ps:
	docker-compose ps

down: clean
	docker-compose down

# create required certificates and keys
certs: SHELL:=/bin/bash
certs:
	yes | ssh-keygen -t rsa -f core/shield/id_rsa -N '' >/dev/null
	cp core/shield/id_rsa.pub agent/agent/authorized_keys
	# crate CA
	openssl req -x509 -newkey rsa:2048 -sha256 -nodes -keyout core/shield/CA.key -out core/shield/CA.pem -subj "/CN=vaultca" -days 365
	# create vault certs
	openssl req -newkey rsa:2048 -sha256 -nodes -keyout vault/vault/vault.key -out vault/vault/vault.csr -subj "/CN=*"
	openssl x509 -days 3065 -sha256 -req -in vault/vault/vault.csr -CA core/shield/CA.pem -CAkey core/shield/CA.key -set_serial 01 -out vault/vault/vault.crt -extfile <(printf "subjectAltName = DNS:localhost,DNS:vault,IP:127.0.0.1,IP:0.0.0.0")
	# create nginx certs
	openssl req -newkey rsa:2048 -sha256 -nodes -keyout nginx/nginx.key -out vault/vault/vault.csr -subj "/CN=*"
	openssl x509 -days 3065 -sha256 -req -in vault/vault/vault.csr -CA core/shield/CA.pem -CAkey core/shield/CA.key -set_serial 01 -out nginx/nginx.crt -extfile <(printf "subjectAltName = DNS:localhost,DNS:nginx,IP:127.0.0.1,IP:0.0.0.0")
	# copy to agent
	cp core/shield/CA.pem agent/agent/CA.pem
	# create key for agent
	yes | ssh-keygen -t rsa -f agent/agent/id_rsa -N '' >/dev/null

clean:
	@rm -rf vault/vault/storage/*
