# SPECIFICATION.md — Fireplace (service map)

> **Monorepo root spec = service map only.** This file lists the services and points
> at where each one's truth lives. **Capability lines live per-service**, never here —
> see each `services/<name>/SPECIFICATION.md` (Tier-1 thin index) and `docs/specs/`
> (Tier-2 `FS-NNNN` feature specs). Platform coordination lives in `CLAUDE.md`.

## Services

| Service              | Bounded context                        | Thin spec (capability index)                                                                     |
| -------------------- | -------------------------------------- | ------------------------------------------------------------------------------------------------ |
| api-gateway          | Gateway / Edge (Notes, User Analytics) | [services/api-gateway/SPECIFICATION.md](services/api-gateway/SPECIFICATION.md)                   |
| auth-service         | Auth                                   | [services/auth-service/SPECIFICATION.md](services/auth-service/SPECIFICATION.md)                 |
| plan-service         | Plan                                   | [services/plan-service/SPECIFICATION.md](services/plan-service/SPECIFICATION.md)                 |
| calendar-service     | Calendar                               | [services/calendar-service/SPECIFICATION.md](services/calendar-service/SPECIFICATION.md)         |
| insights-service     | Insights                               | [services/insights-service/SPECIFICATION.md](services/insights-service/SPECIFICATION.md)         |
| client               | Frontend (Next.js)                     | [client/SPECIFICATION.md](client/SPECIFICATION.md)                                               |

## Shared code

`common/` (and other shared libraries) get **no spec of their own**. A behavior change
in shared code belongs to the `FS-NNNN` of the driving feature in the service that
motivates it.
