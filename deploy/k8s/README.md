# Kubernetes deployment example

Minimum-viable Kubernetes manifests for self-hosting Knowledge Layer. Treat this as a **starting point** — every team customizes ingress, observability, secret management, and resource sizing.

The manifests assume:

- Postgres, Redis, and OpenSearch run **outside** the cluster (managed services or separate stateful workloads). The example does not provision them — point your `DATABASE_URL`, `REDIS_URL`, `OPENSEARCH_URL` at existing endpoints.
- Container images for API and Web are published somewhere reachable by your nodes (GitHub Container Registry, ECR, GCR, in-cluster registry…). Update the `image:` fields in each `*-deployment.yaml`.
- TLS termination happens at your ingress controller, not in-pod. The provided `ingress.example.yaml` is a sketch — adapt to your controller (nginx, Traefik, GKE, ALB, …).

## Files

| File | What it provisions |
|---|---|
| [`namespace.yaml`](namespace.yaml) | A `knowledge-layer` namespace. |
| [`configmap.yaml`](configmap.yaml) | Non-secret env: `APP_ENV`, `APP_PUBLIC_URL`, `OPENSEARCH_URL`, `JOBWORKER_HEALTH_PORT`, `CONNECTORWORKER_HEALTH_PORT`. |
| [`secret.example.yaml`](secret.example.yaml) | **Template** — fill in real values then `kubectl create secret generic` (or use external-secrets / SOPS). Lists every required + optional secret with comments. Do NOT commit a populated copy. |
| [`api-deployment.yaml`](api-deployment.yaml) | API Deployment (2 replicas, rolling update, `/health` probes) + ClusterIP Service. |
| [`jobworker-deployment.yaml`](jobworker-deployment.yaml) | Job worker Deployment + ClusterIP Service for the health endpoint. |
| [`connectorworker-deployment.yaml`](connectorworker-deployment.yaml) | Connector worker Deployment + ClusterIP Service for the health endpoint. |
| [`web-deployment.yaml`](web-deployment.yaml) | Next.js web Deployment + ClusterIP Service. |
| [`ingress.example.yaml`](ingress.example.yaml) | Example Ingress routing `kl.example.com` → API + Web. |

## Apply

Order matters because Deployments reference the ConfigMap and Secret:

```bash
# 1. Create the namespace
kubectl apply -f namespace.yaml

# 2. Provide secrets (one of):
#    a) kubectl create secret generic from CLI flags (test/dev only):
kubectl -n knowledge-layer create secret generic knowledge-layer-secrets \
  --from-literal=DATABASE_URL='postgres://knowledge:…@db.internal:5432/knowledge?sslmode=require' \
  --from-literal=REDIS_URL='redis://redis.internal:6379/0' \
  --from-literal=SESSION_SECRET="$(openssl rand -hex 32)" \
  --from-literal=OPS_AUTH_TOKEN="$(openssl rand -hex 24)" \
  --from-literal=AI_PRIVACY_VAULT_KEY="$(openssl rand -base64 32)" \
  --from-literal=OPENAI_API_KEY='sk-…' \
  --from-literal=CORS_ALLOW_ORIGINS='https://kl.example.com'
#    b) external-secrets / Sealed Secrets / SOPS — recommended for production.

# 3. Apply non-secret config + workloads
kubectl apply -f configmap.yaml
kubectl apply -f api-deployment.yaml
kubectl apply -f jobworker-deployment.yaml
kubectl apply -f connectorworker-deployment.yaml
kubectl apply -f web-deployment.yaml
kubectl apply -f ingress.example.yaml   # adapt first
```

## Verify

```bash
kubectl -n knowledge-layer get pods -w
kubectl -n knowledge-layer logs deploy/knowledge-api -f
kubectl -n knowledge-layer port-forward svc/knowledge-api 8080:80
curl http://localhost:8080/health
```

For the worker `/ops/health` snapshots (Phase 2.2.1), port-forward the worker services on `:9001` / `:9002` (they expose only the health server, not the asynq listener):

```bash
kubectl -n knowledge-layer port-forward svc/knowledge-jobworker 9001:9001
curl -H "Authorization: Bearer $OPS_AUTH_TOKEN" http://localhost:9001/ops/health
```

## What's NOT in this example

- **Database, Redis, OpenSearch** — out of scope. Use a managed offering or a separate Helm chart.
- **HorizontalPodAutoscaler** — opinionated; add per worker family based on `asynq_queue_pending` (Phase 2.2.2 metric).
- **PodDisruptionBudget** — recommended once you scale beyond 2 replicas.
- **NetworkPolicy** — the production hardening doc requires Postgres / Redis / OpenSearch to be private; enforce that with NetworkPolicy or VPC routing.
- **PrometheusServiceMonitor** — depends on your Prometheus operator. The relevant scrape targets are `:8080/metrics` (API), `:9001/ops/health` (jobworker), `:9002/ops/health` (connectorworker), all bearer-gated outside `APP_ENV=local`.
- **TLS certs** — leave to cert-manager or your platform.
- **Helm chart** — these flat manifests are the contract; a Helm chart can wrap them later without changing the underlying shape.

## Production checklist

Before pointing real users at the cluster, satisfy [docs/PRODUCTION_HARDENING.md](../../docs/PRODUCTION_HARDENING.md) and [docs/PRODUCTION_GO_LIVE_CHECKLIST.md](../../docs/PRODUCTION_GO_LIVE_CHECKLIST.md):

- [ ] `APP_ENV=production` in the ConfigMap.
- [ ] All required secrets present in the Secret resource (the API and workers fail-start otherwise — `ValidateAPI` / `ValidateWorker`).
- [ ] `APP_PUBLIC_URL` is `https://…`.
- [ ] `OPENSEARCH_URL` is `https://…` (or `OPENSEARCH_ALLOW_INSECURE_HTTP=1` is consciously set for a private mesh).
- [ ] `AI_PRIVACY_VAULT_KEY` set and `AI_PRIVACY_DEV_PLAINTEXT_STORE` unset.
- [ ] Liveness + readiness probes pass against `GET /health` from inside the namespace.
- [ ] Ingress terminates TLS, sets `X-Forwarded-Proto: https`, and points `kl.example.com/api/*` at the API service while serving the rest from the Web service.
- [ ] Backup procedure for the (out-of-cluster) Postgres is in place — see [docs/UPGRADE_AND_ROLLBACK.md](../../docs/UPGRADE_AND_ROLLBACK.md).

## Related

- Bare-metal/systemd: [docs/SELF_HOSTED.md](../../docs/SELF_HOSTED.md) §"Bare-metal / VM".
- DigitalOcean App Platform: [`../do/`](../do/), [docs/DO_DEPLOYMENT.md](../../docs/DO_DEPLOYMENT.md).
- Compose (local + small single-host prod): [`../../docker-compose.yml`](../../docker-compose.yml).
