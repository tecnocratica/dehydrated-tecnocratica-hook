# dehydrated-tecnocratica-hook

A [dehydrated](https://github.com/dehydrated-io/dehydrated) hook for DNS-01 ACME challenges using the Tecnocratica DNS API (Neodigit / Virtualname).

This hook automates the creation and cleanup of `_acme-challenge` TXT records required for DNS-01 validation, and waits for DNS propagation before returning control to dehydrated.

## Features

- Creates and deletes ACME challenge TXT records via the Tecnocratica API
- Verifies DNS propagation across multiple resolvers before returning
- Exponential backoff polling to avoid excessive API/DNS queries
- Supports `HOOK_CHAIN` (multiple challenge triplets in a single call)
- Supports wildcard certificates (`*.example.com`)
- Structured logging (text or JSON) to stderr

## Installation

### Download a release

Download the latest binary from the [Releases](https://github.com/tecnocratica/dehydrated-tecnocratica-hook/releases) page.

Pre-built binaries are available for:
- Linux (amd64, arm64, 386, armv5, armv6, armv7)
- macOS (amd64, arm64)
- FreeBSD (amd64, arm64)

```bash
# Example for Linux amd64
tar xzf dehydrated-tecnocratica-hook_*_linux_amd64.tar.gz
chmod +x dehydrated-tecnocratica-hook
sudo mv dehydrated-tecnocratica-hook /usr/local/bin/
```

### Install with Go

Requires Go 1.25 or later.

```bash
go install github.com/tecnocratica/dehydrated-tecnocratica-hook/src/cmd/dehydrated-tecnocratica-hook@latest
```

The binary will be installed to `$GOPATH/bin` (or `$HOME/go/bin` by default). Make sure this directory is in your `PATH`.

### Build from source

Requires Go 1.25 or later.

```bash
git clone https://github.com/tecnocratica/dehydrated-tecnocratica-hook.git
cd dehydrated-tecnocratica-hook
go build -ldflags "-s -w \
  -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev) \
  -X main.commit=$(git rev-parse --short HEAD 2>/dev/null || echo unknown) \
  -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o dehydrated-tecnocratica-hook ./src/cmd/dehydrated-tecnocratica-hook
sudo mv dehydrated-tecnocratica-hook /usr/local/bin/
```

## Configuration

All configuration is done through environment variables.

| Variable                           | Required | Default                       | Description                                               |
| ---------------------------------- | -------- | ----------------------------- | --------------------------------------------------------- |
| `TECNOCRATICA_API_TOKEN`           | **Yes**  | -                             | API authentication token for Neodigit/Virtualname         |
| `TECNOCRATICA_API_URL`             | No       | `https://api.neodigit.net/v1` | API base URL                                              |
| `TECNOCRATICA_DNS_RESOLVERS`       | No       | `8.8.8.8,1.1.1.1`             | Comma-separated DNS resolver addresses                    |
| `TECNOCRATICA_PROPAGATION_TIMEOUT` | No       | `120`                         | Maximum seconds to wait for DNS propagation               |
| `TECNOCRATICA_POLL_INTERVAL`       | No       | `5`                           | Initial poll interval in seconds (grows 1.5x per attempt) |
| `TECNOCRATICA_LOG_LEVEL`           | No       | `info`                        | Log level: `debug`, `info`, `warn`, `error`               |
| `TECNOCRATICA_LOG_FORMAT`          | No       | `text`                        | Log format: `text` or `json`                              |

### API URL by provider

| Provider    | API URL                                 |
| ----------- | --------------------------------------- |
| Neodigit    | `https://api.neodigit.net/v1` (default) |
| Virtualname | `https://api.virtualname.net/v1`        |

## Quick start

A complete example from installation to issuing a certificate:

```bash
# 1. Install the hook
go install github.com/tecnocratica/dehydrated-tecnocratica-hook/src/cmd/dehydrated-tecnocratica-hook@latest

# 2. Set your API token
export TECNOCRATICA_API_TOKEN="your-api-token-here"

# 3. Configure dehydrated (/etc/dehydrated/config)
cat >> /etc/dehydrated/config <<EOF
CHALLENGETYPE="dns-01"
HOOK="$(which dehydrated-tecnocratica-hook)"
HOOK_CHAIN="yes"
EOF

# 4. Register and issue a certificate
dehydrated --register --accept-terms
dehydrated -c -d example.com
```

For Virtualname users, also set the API URL before running dehydrated:

```bash
export TECNOCRATICA_API_URL="https://api.virtualname.net/v1"
```

## Usage with dehydrated

### 1. Set up environment variables

Export your API token (and any other overrides) in your shell or in a wrapper script:

```bash
export TECNOCRATICA_API_TOKEN="your-api-token-here"
```

### 2. Configure dehydrated

In your dehydrated config file (usually `/etc/dehydrated/config`), set the hook and challenge type:

```bash
CHALLENGETYPE="dns-01"
HOOK="/usr/local/bin/dehydrated-tecnocratica-hook"

# Optional: enable hook chaining for faster multi-domain certificates
HOOK_CHAIN="yes"
```

### 3. Run dehydrated

```bash
# Register account (first time only)
dehydrated --register --accept-terms

# Issue or renew certificates
dehydrated -c
```

### Example: single domain

```bash
dehydrated -c -d example.com
```

### Example: SAN certificate (multiple domains)

```bash
dehydrated -c -d example.com -d www.example.com -d mail.example.com
```

### Example: wildcard certificate

```bash
dehydrated -c -d '*.example.com'
```

### Wrapper script

If you prefer a self-contained wrapper script instead of exporting variables:

```bash
#!/bin/bash
# /etc/dehydrated/hooks/tecnocratica.sh

export TECNOCRATICA_API_TOKEN="your-api-token-here"
# export TECNOCRATICA_API_URL="https://api.virtualname.net/v1"  # for Virtualname

exec /usr/local/bin/dehydrated-tecnocratica-hook "$@"
```

Then point dehydrated at the wrapper:

```bash
HOOK="/etc/dehydrated/hooks/tecnocratica.sh"
```

## How it works

When dehydrated invokes the hook, the following happens:

1. **`deploy_challenge`** -- The hook creates `_acme-challenge.<domain>` TXT records via the API, then polls the configured DNS resolvers with exponential backoff until all records are found (or the propagation timeout is reached).

2. **`clean_challenge`** -- The hook deletes the TXT records that were created.

3. **`startup_hook` / `exit_hook`** -- Informational logging only.

4. **Other hooks** (`deploy_cert`, `unchanged_cert`, etc.) -- No-ops; the hook exits successfully.

All log output goes to **stderr**, keeping stdout clean for dehydrated.

## Troubleshooting

### Enable debug logging

```bash
export TECNOCRATICA_LOG_LEVEL=debug
dehydrated -c -d example.com
```

This shows detailed information about API calls, DNS lookups, and propagation polling.

### Propagation timeout

If you get `propagation timeout` errors, your DNS records may be slow to propagate. Try increasing the timeout:

```bash
export TECNOCRATICA_PROPAGATION_TIMEOUT=300
```

### Custom DNS resolvers

If the default resolvers (Google and Cloudflare) are blocked or unreliable in your environment:

```bash
export TECNOCRATICA_DNS_RESOLVERS="9.9.9.9,208.67.222.222"
```

### Virtualname users

If your DNS is managed by Virtualname instead of Neodigit, set the API URL:

```bash
export TECNOCRATICA_API_URL="https://api.virtualname.net/v1"
```

## License

MIT -- see [LICENSE](LICENSE).
