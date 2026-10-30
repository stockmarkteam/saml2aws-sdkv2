# saml2aws-sdkv2

An unofficial fork of [Versent/saml2aws](https://github.com/Versent/saml2aws) that replaces the end-of-support
AWS SDK for Go v1 with **[AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2)** and builds with a current Go toolchain (**Go 1.27.3**, minimum **Go 1.26.10**).

Commands, identity providers, configuration files and flags are unchanged. If you already use saml2aws, this is a drop-in replacement. The few behaviour differences are listed [below](#behaviour-differences-from-upstream); please read them before you roll this out.

> **Not affiliated with Versent.** This fork is maintained separately. Please report problems with this build
> [here](https://github.com/stockmarkteam/saml2aws-sdkv2/issues), not on the upstream tracker.

- [Why this fork](#why-this-fork)
- [What changed](#what-changed)
- [Behaviour differences from upstream](#behaviour-differences-from-upstream)
- [Install](#install)
- [Build from source](#build-from-source)
- [Usage](#usage)
- [Versioning](#versioning)
- [License](#license)

## Why this fork

- **AWS SDK for Go v1 reached end-of-support on July 31, 2025.** It no longer gets bug fixes or security updates.
  saml2aws only uses it for AWS STS calls, so the migration is small. This fork does that migration.
- **The Go toolchain is current.** Upstream declares Go 1.21 with toolchain 1.22.5, so its builds miss two years of
  standard-library security fixes (TLS, `net/http`, `crypto/*`).

## What changed

Based on upstream `master` at [`65195ee`](https://github.com/Versent/saml2aws/commit/65195ee65d15) (after v2.36.19).

| Area | Upstream | This fork |
|---|---|---|
| AWS SDK | `github.com/aws/aws-sdk-go` v1.55.6 | `github.com/aws/aws-sdk-go-v2` v1.47 (`config`, `service/sts`, `credentials`) |
| Go | `go 1.21.0`, `toolchain go1.22.5` | `go 1.26.10`, `toolchain go1.27.3` |
| Lint | golangci-lint v1.55.2 | golangci-lint v2.15.0 (config migrated to the v2 format) |
| Build container | `Dockerfile.build`: `ubuntu:jammy` + Docker CLI, host Docker socket mounted, buildx | `golang:1.27.3-trixie` + GoReleaser 2.19.0, no socket or buildx; works with rootless Podman |

The code changes are limited to the STS calls in `cmd/saml2aws/commands/login.go` (`AssumeRoleWithSAML`) and
`cmd/saml2aws/commands/exec.go` (`GetCallerIdentity`, profile-based role assumption for `exec --exec-profile`).
There's also one `go vet` fix in the Shibboleth provider. Identity-provider code is untouched.

## Behaviour differences from upstream

### 1. STS endpoint: regional instead of global

| Region configured? | Upstream (SDK v1) | This fork (SDK v2) |
|---|---|---|
| No region anywhere | `https://sts.amazonaws.com` | `https://sts.amazonaws.com` (same) |
| A region, e.g. `ap-northeast-1` | `https://sts.amazonaws.com` for most regions\* | `https://sts.ap-northeast-1.amazonaws.com` |

\* SDK v1 used the global endpoint for `us-east-1` and the other long-standing regions unless you set `AWS_STS_REGIONAL_ENDPOINTS=regional`.

The region comes from, in order: `--region` / `SAML2AWS_REGION`, `region` in `~/.saml2aws`, `AWS_REGION` / `AWS_DEFAULT_REGION`,
and then `region` in the shared AWS config (`~/.aws/config`).

What this means for you:

- **Firewalls and proxies:** if outbound traffic is allow-listed, permit `sts.<region>.amazonaws.com` (or
  `sts.<region>.amazonaws.com.cn` in China regions) as well as `sts.amazonaws.com`. This is the most likely cause
  of a working upstream setup failing with this fork.
- **STS must be active in that region.** It is by default, but an administrator can deactivate a regional STS
  endpoint in IAM account settings.
- **SigV4A works out of the box.** The upstream caveat about setting `AWS_STS_REGIONAL_ENDPOINTS=regional` no longer
  applies. SDK v2 ignores that variable.
- To keep using the global endpoint, leave the region unset everywhere in the list above.

### 2. `exec --exec-profile` always reads `~/.aws/config`

SDK v1 only loaded the shared config file when `AWS_SDK_LOAD_CONFIG` was set. SDK v2 always loads it, so
`role_arn` / `source_profile` chains work without that variable.

### 3. Errors while checking existing credentials

`saml2aws exec` checks the saved credentials with `GetCallerIdentity`. If they have expired, or the profile hasn't been
created yet, it logs in again, the same as upstream. Other credential errors (for example a malformed
`~/.aws/config`) are now reported instead of silently starting a new login.

### 4. Go 1.26 TLS defaults

Building with a current Go version turns on its stricter TLS defaults: TLS 1.2 minimum, no RSA key exchange and no 3DES.
Modern identity providers aren't affected. If an old on-premises ADFS or Shibboleth server stops working, rebuild with
a `godebug` block in `go.mod`, for example:

```
godebug (
	tls10server=1
	tlsrsakex=1
)
```

Unchanged: `--session-duration` still overrides `duration_seconds` in an AWS profile, and the credentials file,
`~/.saml2aws` and the keychain/credential-store entries keep the same names and format.

## Install

Pre-built binaries are attached to each [release](https://github.com/stockmarkteam/saml2aws-sdkv2/releases):

| File | Platform |
|---|---|
| `saml2aws_<version>_linux_amd64.tar.gz` | Linux x86-64 (static) |
| `saml2aws_<version>_linux_arm64.tar.gz` | Linux ARM64 (static) |
| `saml2aws_<version>_windows_amd64.zip` | Windows x64 |
| `saml2aws_<version>_windows_arm64.zip` | Windows ARM64 |
| `saml2aws-<version>-src.tar.gz` | Source |
| `SHA256SUMS` | Checksums for all of the above |

These binaries are built without cgo, the same way as upstream's static Linux and Windows builds. They **don't support USB security keys (U2F/FIDO via HID)**. If you need one, [build from source](#build-from-source) with
`CGO_ENABLED=1`.

There's no macOS binary: macOS Keychain support needs cgo, so it can't be cross-compiled. Build it on a Mac (see below).

### Linux

```sh
VERSION=2.36.19-sdkv2.1
curl -LO https://github.com/stockmarkteam/saml2aws-sdkv2/releases/download/v${VERSION}/saml2aws_${VERSION}_linux_amd64.tar.gz
curl -LO https://github.com/stockmarkteam/saml2aws-sdkv2/releases/download/v${VERSION}/SHA256SUMS
sha256sum --ignore-missing -c SHA256SUMS
tar -xzf saml2aws_${VERSION}_linux_amd64.tar.gz saml2aws
sudo install -m 0755 saml2aws /usr/local/bin/saml2aws
saml2aws --version
```

### Windows (PowerShell)

```powershell
$Version = "2.36.19-sdkv2.1"
Invoke-WebRequest "https://github.com/stockmarkteam/saml2aws-sdkv2/releases/download/v$Version/saml2aws_${Version}_windows_amd64.zip" -OutFile saml2aws.zip
(Get-FileHash saml2aws.zip -Algorithm SHA256).Hash   # compare with SHA256SUMS
Expand-Archive saml2aws.zip -DestinationPath "$env:LOCALAPPDATA\saml2aws"
# add %LOCALAPPDATA%\saml2aws to your PATH
```

### Replacing an existing upstream install

Remove or unlink the upstream binary first so the right one is on your `PATH`
(`brew uninstall saml2aws`, `choco uninstall saml2aws`, or delete the old binary). Your existing configuration and
saved credentials keep working.

## Build from source

Requirements:

- Go **1.26.10 or newer**. `go.mod` also sets `toolchain go1.27.3`, so with the default `GOTOOLCHAIN=auto` an older 1.26.x
  will download Go 1.27.3 before building. On offline or air-gapped machines, set `GOTOOLCHAIN=local` to build with
  the installed Go (1.26.10 or newer).
- For cgo builds on Debian/Ubuntu: `sudo apt install build-essential libudev-dev`.

```sh
tar -xzf saml2aws-2.36.19-sdkv2.1-src.tar.gz && cd saml2aws   # or: git clone https://github.com/stockmarkteam/saml2aws-sdkv2 && cd saml2aws-sdkv2
VERSION=2.36.19-sdkv2.1

# Linux / Windows, static (no USB security keys)
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.Version=${VERSION}" -o saml2aws ./cmd/saml2aws

# Linux with USB security-key support
CGO_ENABLED=1 go build -trimpath -tags hidraw -ldflags "-s -w -X main.Version=${VERSION}" -o saml2aws ./cmd/saml2aws

# macOS (Keychain support needs cgo, and Xcode command-line tools)
CGO_ENABLED=1 go build -trimpath -ldflags "-s -w -X main.Version=${VERSION}" -o saml2aws ./cmd/saml2aws
```

Without `-X main.Version=...`, `saml2aws --version` prints the placeholder `1.0.0`. To check what a binary was built
with, whatever its version string says:

```sh
go version -m ./saml2aws    # shows the Go version and exact module versions, including aws-sdk-go-v2
```

Run the tests with `go test ./...`. The `pkg/provider/browser` tests download a Playwright driver and fail without
internet access, as they do upstream.

### Build in a container (Podman or Docker)

If you don't want to install Go and GoReleaser locally, the Makefile can build inside a container. It uses
`podman` if it's installed, otherwise `docker` (override with `CONTAINER_ENGINE=docker`). `Dockerfile.build` uses
plain Dockerfile syntax with no BuildKit or buildx features, and no container-engine socket is mounted.

```sh
make build-image          # build localhost/saml2aws-build from Dockerfile.build
make build-in-container   # goreleaser snapshot build into ./dist (needs a git checkout)
make test-in-container    # go test ./...
make build-environment    # interactive shell with Go, gcc, libudev and goreleaser
make container-image      # localhost/saml2aws:snapshot from dist/, built without buildx
```

Files written to `./dist` are owned by you: rootless Podman maps container root to your user, and with Docker the
Makefile passes `--user`. Go module and build caches persist in the `saml2aws-build-cache` volume. Publishing
multi-arch images to a registry is still done by CI through GoReleaser.

## Usage

Usage is the same as upstream. For details, refer to the upstream README **[README.upstream.md](https://github.com/Versent/saml2aws/blob/master/README.md**. It covers every identity provider, `configure`, `login`, `exec`, `console`, `script`, multi-account setups and credential-process integration. Its install section points at upstream releases, so use the [Install](#install) section above instead.

Quick start:

```sh
saml2aws configure                  # set up an IdP account in ~/.saml2aws
saml2aws login                      # authenticate and write temporary credentials to ~/.aws/credentials
saml2aws exec -- aws sts get-caller-identity
```

## Versioning

Releases are tagged `v<upstream version>-sdkv2.<n>`, for example `v2.36.19-sdkv2.1`: the upstream release the
fork is based on, followed by this fork's own revision number. `<n>` increases for fork-only changes such as
dependency or toolchain updates and fixes.

## License

MIT, the same as upstream. The original code is Copyright (c) 2024 [Versent](https://versent.com.au); see [LICENSE.md](https://github.com/Versent/saml2aws/blob/master/LICENSE.md). Changes in this fork are released under the same license. "saml2aws" is used here only to identify the upstream project this fork is based on.
