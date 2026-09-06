# Troubleshooting

## The browser reports that the API is unavailable

Confirm that `GET /healthz` returns HTTP 200 and that the configured port is not already in use. The default address is `127.0.0.1:8080`; use `--address` or `PLATFORM_ADDRESS` to change it.

## Token mode exits during startup

`PLATFORM_API_TOKEN` must contain at least 16 characters. Verify that the variable is available to the API process and that `PLATFORM_AUTH_ROLE` names a supported role.

## Requests return 401

In demo mode, include `X-Actor` and `X-Role`. In token mode, include `Authorization: Bearer TOKEN` and `X-Actor`. The `platformctl` client adds these headers from its flags or environment variables.

## Requests return 403

The authenticated role does not have permission for that operation. Developers cannot read audit events; auditors cannot create or retry services. In token mode the API uses the server-configured role, not the request's role header.

## A service already exists

Service names are immutable catalogue identities. Choose a new name. Repeating the original descriptor with its original idempotency key is safe and returns the original operation.

## An operation failed

Read it with `platformctl operation OPERATION_ID`. Remove the reported cause, then run `platformctl retry --reason "CAUSE REMOVED" OPERATION_ID`. Only failed operations can be retried. See the detailed operation runbook.

## Docker Compose cannot write state

The container runs as a non-root user and stores data in the managed `platform-data` volume. Remove unsupported bind-mount overrides or ensure the mounted directory is writable by UID and GID 65532.

## Helm creates a pending pod

Check the PersistentVolumeClaim and the cluster's default StorageClass. For an ephemeral evaluation, set `persistence.enabled=false`. Keep `replicaCount=1` with the included file store.
