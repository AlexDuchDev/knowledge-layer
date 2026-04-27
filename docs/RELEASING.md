# Releasing (OSS and self-host)

## Versioning

- Use [Semantic Versioning](https://semver.org/) (`MAJOR.MINOR.PATCH`).
- Keep [CHANGELOG.md](../CHANGELOG.md) updated under **`[Unreleased]`** during development; at release time rename the section to the new version and date.

## Pre-release checks

From repository root (`Knowledge Layer Local/`):

```bash
./scripts/repo-sanity-check.sh
cd apps/api && go test ./... -count=1
cd ../web && npm ci && npm run build
```

For Docker images: see [.github/workflows/docker.yml](../.github/workflows/docker.yml).

## Git tag and GitHub Release (automated)

1. Merge to the default branch with CI green.
2. **Update CHANGELOG.md**: rename the `[Unreleased]` heading to `## [v0.X.Y] — YYYY-MM-DD` so the release workflow can extract the body. Open a small commit just for this rename if it makes the diff clearer.
3. Tag and push:
   ```bash
   git tag -a v0.X.Y -m "Release v0.X.Y"
   git push origin v0.X.Y
   ```
4. The [`Release` workflow](../.github/workflows/release.yml) fires automatically on the tag push and:
   - Re-runs `make lint typecheck`, `go test ./...`, and `npm run build` on the tagged commit (a tag pointing at an old SHA could be ungreen — we re-verify before publishing).
   - Builds and pushes multi-arch (`linux/amd64` + `linux/arm64`) images to GitHub Container Registry:
     - `ghcr.io/<owner>/knowledge-layer-api:vX.Y.Z` (and `:latest` on stable releases)
     - `ghcr.io/<owner>/knowledge-layer-web:vX.Y.Z` (and `:latest` on stable releases)
   - Creates the GitHub Release with the matching CHANGELOG section as the body.
5. **Pre-releases** (`vX.Y.Z-rcN`, `-beta1`, etc.) are tagged automatically as GitHub pre-releases and do **not** receive the `:latest` tag.

If you need to re-publish without a re-tag (e.g. amending the build), the workflow has no manual trigger by design — re-tag with a patch bump (`v0.X.Y+1`) and let the chain run again. This keeps every published image traceable to a unique tag.

## Container image consumption

Operators reference the published images directly:

```bash
docker pull ghcr.io/<owner>/knowledge-layer-api:v0.X.Y
docker pull ghcr.io/<owner>/knowledge-layer-web:v0.X.Y
```

For Kubernetes, update the `image:` field in [`deploy/k8s/*-deployment.yaml`](../deploy/k8s/) to the published tag. For the bare-metal/systemd path see [SELF_HOSTED.md](./SELF_HOSTED.md) §"Bare-metal / VM".

## OSS contract

After behavior or surface changes, update [OSS_V1_SCOPE.md](./OSS_V1_SCOPE.md) and [LIMITATIONS.md](./LIMITATIONS.md) together per [DOCS_MAINTENANCE_POLICY.md](./DOCS_MAINTENANCE_POLICY.md).
