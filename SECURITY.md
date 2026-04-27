# Security policy

## Supported versions

Security fixes are applied to the default branch. Self-hosted operators should track releases and upgrade regularly.

## Reporting a vulnerability

Please report security issues **privately** (do not open a public issue with exploit details).

1. **Preferred (GitHub):** [Security advisories](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability) — *Private vulnerability reporting* on the canonical repository (enable it in **Settings → Security → Code security** if you are publishing a fork).
2. **Email:** maintainers should publish a contact address in the GitHub org or repository profile; until then, use the advisory flow above.
3. Include: affected component (`apps/api`, `apps/web`, etc.), reproduction steps, and impact assessment if known.

Before publishing a release, run [`scripts/repo-sanity-check.sh`](scripts/repo-sanity-check.sh) from the monorepo root (see [docs/RELEASING.md](docs/RELEASING.md)).

We aim to acknowledge reports within a few business days.

## Scope notes

- This project is intended for **self-hosted** deployment. Operators are responsible for network isolation, TLS, secrets management, and database backups.
- Pilot authentication via `X-Principal-User-ID` is for **development only**; production deployments should use session-based auth (see `AUTH_MODE` in [docs/SELF_HOSTED.md](docs/SELF_HOSTED.md)).
