# Configuration

The API accepts command-line flags and environment variables. Flags take precedence.

| API setting | Flag | Environment variable | Default |
| --- | --- | --- | --- |
| Listen address | `--address` | `PLATFORM_ADDRESS` | `127.0.0.1:8080` |
| State file | `--data` | `PLATFORM_DATA_PATH` | `.platform/state.json` |
| Generated repositories | `--generated-root` | `GENERATED_SERVICES_DIR` | `.platform/generated` |
| Authentication mode | `--auth-mode` | `PLATFORM_AUTH_MODE` | `demo` |
| Token role | `--token-role` | `PLATFORM_AUTH_ROLE` | `platform_admin` |
| Bearer token | none | `PLATFORM_API_TOKEN` | empty |
| Fault injection step | none | `PLATFORM_FAIL_AT_STEP` | empty |

`PLATFORM_AUTH_MODE` accepts `demo` or `token`. Token mode refuses to start when `PLATFORM_API_TOKEN` has fewer than 16 characters. `PLATFORM_AUTH_ROLE` must be one of `developer`, `service_owner`, `platform_admin`, or `auditor`.

The CLI supports these settings on every command:

| CLI flag | Environment variable | Default |
| --- | --- | --- |
| `--address` | `PLATFORM_API_URL` | `http://127.0.0.1:8080` |
| `--actor` | `PLATFORM_ACTOR` | Current operating-system user |
| `--role` | `PLATFORM_ROLE` | `developer` |
| `--token` | `PLATFORM_API_TOKEN` | empty |

Prefer the environment variable for tokens because command-line arguments may be visible in process listings and shell history.

## Shared starter configuration

Generate a high-entropy token with your approved secret manager, then start:

```bash
PLATFORM_AUTH_MODE=token \
PLATFORM_AUTH_ROLE=platform_admin \
PLATFORM_API_TOKEN='replace-with-a-random-secret' \
platform-api --address 0.0.0.0:8080
```

Clients set the same token through `PLATFORM_API_TOKEN`. The server assigns the configured role; a client's `X-Role` value is ignored in token mode.

## Persistence and backup

Stop the API or take a storage-level consistent snapshot before copying the state file. Back up both the configured state file and generated root. The state file is replaced atomically after every committed transition and is created with owner-only permissions on operating systems that enforce POSIX modes.

The included file store supports one API process. Do not point two replicas at the same file. Use the schema under `migrations/` when implementing the PostgreSQL store for high availability.
