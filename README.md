# homelabc

Install without root:

```sh
./install.sh
```

The default destination is `~/.local/bin`; `INSTALL_DIR` overrides it.

## Interactive management shell

```sh
homelabc run \
  --image registry.ryuugu.dev/arch-provisioner:latest \
  --stigmergy-api-url https://stigmergy.ryuugu.dev \
  --inventory-capture-group ssh-managed \
  --api-token-file /absolute/path/to/admin-token
```

The CLI mounts only the token file read-only. Normal container startup fetches
the existing `SSHKeyPair/ansible-runner`, its owned private/public-key Secret,
and the current `SSHCertificate/ansible-runner`. It validates the identity and
writes private container-local credentials, then derives strict host trust from
the API. It never generates keys or obtains the CA private key.

The token must authorize these reads; the admin token works with the current
policy. Missing permissions, unready/replaced resources, invalid keys or expired
certificates stop startup. Credentials disappear when the disposable container
is removed. Start a fresh shell to get a renewed certificate; there is no
background refresh in a running shell.

Settings can instead live under `general` in `~/.homelabc.yaml`:

```yaml
general:
  image: registry.ryuugu.dev/arch-provisioner:latest
  stigmergy_url: https://stigmergy.ryuugu.dev
  inventory_capture_group: ssh-managed
  api_token_file: /absolute/path/to/admin-token
```

The token file must be readable by container UID 1000. An optional
`--ssh-known-hosts-file` supports explicitly trusted non-Server administrative
inventories; ordinary managed Servers use API-derived trust.

The normal-shell startup logic is in `arch-provisioner/profile.d/init.sh`; the
typed credential loader is `ansible-roles/operators/runner_credentials.py`.
Rebuild/publish the runner image after startup changes and reinstall the CLI
after CLI changes. `homelabc run` does not build images or rotate credentials.
