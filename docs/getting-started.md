# Getting Started

This guide takes a new user from an empty machine to a generated service repository.

## Option 1: Docker Compose

1. Install Docker Desktop or Docker Engine with Compose v2.
2. Clone the repository and run `docker compose up --build`.
3. Wait for the log entry `platform API listening`.
4. Open `http://127.0.0.1:8080`.
5. Keep the default descriptor and select **Create service**.
6. Confirm that all four operation steps become green and the catalogue shows `READY`.

The API container writes state and generated repositories to the `platform-data` volume. Run `docker compose down` to stop without deleting data. Run `docker compose down --volumes` only when you intentionally want to delete that state.

## Option 2: Release binaries

1. Open the repository's Releases page.
2. Download the archive matching your operating system and CPU architecture.
3. Verify the archive against `checksums.txt`.
4. Extract both `platform-api` and `platformctl` into a directory on your `PATH`.
5. Run `platform-api` and open `http://127.0.0.1:8080`.

Linux example:

```bash
sha256sum --check checksums.txt --ignore-missing
./platform-api
```

PowerShell example:

```powershell
Get-FileHash .\internal-developer-platform_*.zip -Algorithm SHA256
.\platform-api.exe
```

## Create from the CLI

With the API running:

```bash
platformctl create --file examples/payments-notifier.json
```

The command waits for a terminal operation and prints the complete operation. The expected status is `SUCCEEDED`. List persistent records with:

```bash
platformctl list
platformctl operations
platformctl download payments-notifier
```

The download command writes `payments-notifier.zip`, which is especially useful when the platform runs in Docker or Kubernetes and its generated root is stored in a managed volume.

## Run a generated service

Change into `.platform/generated/payments-notifier`, then follow the generated README. For the default Go template:

```bash
go run ./cmd/server
curl http://127.0.0.1:8080/healthz
```

If the platform API is already using port 8080, stop it first or change the generated service port in its descriptor and recreate the service with a new name.

## Next steps

- Enable token mode before allowing another machine to connect.
- Review the generated Kubernetes image reference before applying its manifest.
- Read the operation runbook before testing failure injection.
- Use the PostgreSQL migration and adapter boundaries as the starting point for multi-replica production evolution.
