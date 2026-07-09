# auth-service — Specification

> Scope: user identity, credentials, JWT issuance, profiles. Platform glossary & cross-service maps: ../../CLAUDE.md.

## Domain Terms

- **User** — an account: `id`, `name`, `email` (unique), bcrypt-hashed password, optional `display_name` + `bio`.
- **Access token** — short-lived JWT (type `access`) proving identity on gateway-fronted calls.
- **Refresh token** — medium-lived JWT (type `refresh`) exchanged for a fresh token pair; stateless, no server store.
- **`display_name`** — user-chosen presentation name; falls back to `name` in UI (FE concern).

## Features

### Implemented

- [x] User registration and login with email/password (JWT access + refresh tokens)
- [x] Password hashing with bcrypt
- [x] Refresh-token exchange for a new access/refresh pair
- [x] User CRUD over gRPC: get one, list all, delete
- [x] Profile columns `display_name` + `bio` on `users` (migration `000003`)
- [x] Read profile (`GetUser`) and update profile (`UpdateProfile` — `name`, `display_name`, `bio`)
- [x] `user.created` / `user.deleted` events published to `auth.events`

### In Progress

- [ ] `user.updated` event (routing key reserved; currently a logged stub, no proto event)

### Future

- [ ] Password change / reset flow
- [ ] Refresh-token revocation (would require a server-side store)

> Not in scope (per master spec): avatar, timezone, notifications, public profiles.

## Data Model

Table `users` (Postgres, `fireplace_auth_service_db` @ :5301):

| Column         | Type      | Constraints                          |
| -------------- | --------- | ------------------------------------ |
| `id`           | UUID      | PK, default `uuid_generate_v4()`     |
| `name`         | TEXT      | NOT NULL                             |
| `email`        | TEXT      | UNIQUE, NOT NULL                     |
| `password`     | TEXT      | NOT NULL (bcrypt hash; field `HashedPassword`) |
| `display_name` | TEXT      | nullable                             |
| `bio`          | TEXT      | nullable                             |
| `created_at`   | TIMESTAMP | default `CURRENT_TIMESTAMP`          |
| `updated_at`   | TIMESTAMP | default `CURRENT_TIMESTAMP`          |

No `refresh_tokens` table — refresh tokens are stateless JWTs.

## gRPC Surface

Service `auth.AuthService` (proto `common/api/proto/auth`):

| Method          | Input                  | Output               | Notes                                                        |
| --------------- | ---------------------- | -------------------- | ------------------------------------------------------------ |
| `SignUp`        | `SignUpRequest` (name, email, password) | `AuthResponse` | Creates user, publishes `user.created`, returns token pair.  |
| `SignIn`        | `SignInRequest` (email, password)        | `AuthResponse` | Enumeration-safe; wrong creds → `Unauthenticated`.           |
| `RefreshToken`  | `RefreshTokenRequest` (refresh_token)    | `AuthResponse` | Validates refresh JWT, reloads user, re-issues pair.         |
| `GetUser`       | `GetUserRequest` (id)  | `User`               | Profile read. Invalid UUID → `InvalidArgument`.              |
| `ListUsers`     | `ListUsersRequest`     | `ListUsersResponse`  | All users, newest first; password field blanked.             |
| `UpdateProfile` | `UpdateProfileRequest` (id, name?, display_name?, bio?) | `User` | Partial update; empty `name` rejected; publishes `user.updated` (stub). |
| `DeleteUser`    | `DeleteUserRequest` (id) | `google.protobuf.Empty` | Deletes user, publishes `user.deleted`. Missing → `NotFound`. |

`AuthResponse` = `{ user, access_token, refresh_token, access_expires_in, refresh_expires_in }` (expiries in nanoseconds).

## Events

Exchange `auth.events` (topic, durable). All published messages are protobuf, persistent.

| Event          | Routing key    | Payload             | Trigger        |
| -------------- | -------------- | ------------------- | -------------- |
| User created   | `user.created` | `UserCreatedEvent`  | `SignUp`       |
| User deleted   | `user.deleted` | `UserDeletedEvent`  | `DeleteUser`   |
| User updated   | `user.updated` | *(none — logged stub)* | `UpdateProfile` (not published) |

**Consumed:** none. Queue `auth-service.events` is declared with a no-op consumer stub — auth-service is upstream-only today.

## Business Rules

- **JWT lifecycle.** On sign-up, sign-in, and refresh the service issues an `access` + `refresh` JWT pair signed with the shared `JWT_SECRET`. Actual TTLs come from env (`ACCESS_TOKEN_TTL` / `REFRESH_TOKEN_TTL`); the deployed `.env` sets **access = 24h (1 day)** and **refresh = 168h (7 days)**. Code-const fallbacks (if env unset/unparseable): access `1h`, refresh `168h`. Refresh validates the incoming refresh token's signature + type, then reloads the user and issues a new pair.
- **bcrypt.** Passwords hashed with `bcrypt.DefaultCost` on sign-up; verified with constant-time compare on sign-in. Hash never leaves the service (JSON `-`; list query blanks it).
- **Account-enumeration-safe sign-in.** A non-existent email and a wrong password both return the same `Unauthorized` result — no distinction is exposed.
- **Cascade via events.** Deleting a user emits `user.deleted`; downstream services (plan, calendar, insights, etc.) consume it to clean up their own owned data. auth-service does not reach into other domains.

## Edge Cases (BE behavior)

| Scenario                        | Behavior                                                  |
| ------------------------------- | -------------------------------------------------------- |
| Sign up with existing email     | Unique-constraint violation → domain conflict error → gRPC error. |
| Sign up missing name/email/password | `ErrInvalidInput` → `InvalidArgument`.               |
| Sign in, unknown email          | `Unauthorized` (indistinguishable from wrong password).  |
| Sign in, wrong password         | `Unauthorized`.                                          |
| Refresh with expired/invalid token | `Unauthorized`.                                       |
| `UpdateProfile` with empty `name` | `ErrInvalidInput` → `InvalidArgument`.                 |
| `UpdateProfile` with no fields  | No-op update; returns current user.                      |
| `GetUser`/`DeleteUser` unknown id | `NotFound`.                                            |
| Malformed UUID                  | `InvalidArgument`.                                       |

## Owned elsewhere

- **FE auth/profile UI** (auth page, profile page, header dropdown, token-lifecycle storage/silent-refresh) → `client/` (flow-client). This spec covers only the BE `GetUser` / `UpdateProfile` surface those pages call.
- **Edge JWT validation & ownership enforcement** (extract `user_id` from JWT `sub`, 401 on missing/expired, 403 on ownership violations, per-user query scoping) → **api-gateway**. auth-service only *issues* tokens; it never validates inbound access tokens.
