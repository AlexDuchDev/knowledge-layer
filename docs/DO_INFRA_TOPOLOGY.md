# DigitalOcean infrastructure topology

Reference **topology** for a production-style deployment (not mandatory—adapt to your scale).

```mermaid
flowchart TB
  subgraph edge [Edge]
    LB[LoadBalancer_TLS]
  end
  subgraph compute [Compute]
    API[API_Container]
    WEB[Web_Container]
    JW[JobWorker]
    CW[ConnectorWorker]
  end
  subgraph data [Data]
    PG[(Postgres_pgvector)]
    RD[(Redis)]
    OS[(OpenSearch_TLS)]
  end
  LB --> API
  LB --> WEB
  API --> PG
  API --> RD
  API --> OS
  JW --> PG
  JW --> RD
  CW --> PG
  CW --> RD
  CW --> OS
```

## Notes

- **API** and **web** terminate TLS at the load balancer or platform edge; set `APP_PUBLIC_URL` and cookie flags per [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md).
- **Workers** are separate processes (same image, different command) sharing `DATABASE_URL`, `REDIS_URL`, and OpenSearch settings.
- **OpenSearch**: use a secured cluster (TLS + auth); do not reuse local compose’s security-disabled profile.

## Related

- [DO_DEPLOYMENT.md](DO_DEPLOYMENT.md)
- [deploy/do/README.md](../deploy/do/README.md)
