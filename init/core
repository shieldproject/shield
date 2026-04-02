#!/bin/bash
set -eu

# Create directories
mkdir -p /shield/data /shield/vault-data /etc/shield

# Generate legacy agent SSH key if not present
if [[ ! -f /etc/shield/ssh.key ]]; then
  rm -f /etc/shield/ssh.key.pub
  ssh-keygen -m PEM -t rsa -f /etc/shield/ssh.key -N ''
  rm -f /etc/shield/ssh.key.pub
fi
export SHIELD_LEGACY_AGENTS_PRIVATE_KEY=$(cat /etc/shield/ssh.key)

# Start Vault in background
echo "$(date) starting vault..."
vault server -config=/shield/config/vault.conf &
VAULT_PID=$!

# Wait for Vault to be listening (SHIELD handles init/unseal itself via the API)
echo "$(date) waiting for vault..."
count=60
while ! curl -so /dev/null http://127.0.0.1:8200/v1/sys/health 2>/dev/null; do
  sleep 1
  count=$((count - 1))
  if [[ $count -eq 0 ]]; then
    echo >&2 "vault failed to start after 60 seconds"
    exit 1
  fi
done
echo "$(date) vault is listening"

# Run SHIELD database schema migrations
echo "$(date) running shield-schema..."
shield-schema -d /shield/data/shield.db

# Start SHIELD daemon
echo "$(date) starting shieldd..."
exec shieldd --log-level debug --config /shield/config/shieldd.conf
