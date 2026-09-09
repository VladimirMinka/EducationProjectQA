# Release notes — Store API Simulator

Changelog for mentors and QA mentees. Newest release first.  
After pulling a release that changes DB schema, recreate the volume:

```bash
docker compose down -v && docker compose up --build
```

API reference: [`Docs.MD`](Docs.MD) · Project overview: [`README.md`](README.md)

---

## Release 0.10.0 — 2026-08-29

**Theme:** Catalog search, filters, categories, pagination

### Added / Changed

- Categories table + `category_id` on products; `GET /v1/categories`, `GET /v1/categories/{id}`
- `ListProducts` filters: `q`, `brand`, `category_id`, `min/max_price_cents`, `in_stock`, `sort`, real `page_size` / `page_token` / `total_count`
- UI catalog uses server-side search/filters + «Показать ещё»
- One seed SKU with `stock_quantity=0` for `in_stock` tests (`…0015`)

### Migrations

- `007_catalog_search.sql`

### Breaking / QA impact

- Default list is paginated (20); request `page_size=50` or paginate for full catalog
- Cover filter combos, bad token, empty result, REST=gRPC

---

## Release 0.9.0 — 2026-08-29

**Theme:** Order status job queue

### Added / Changed

- Postgres `order_jobs` + in-process worker (`ORDER_STATUS_DELAY`, `ORDER_JOB_POLL_INTERVAL`)
- `CREATED→PAID` enqueues `SHIPPED`; worker then enqueues `COMPLETED`
- `GET` / `ListOrders` no longer auto-progress statuses
- Admin `GET /v1/admin/orders/{order_id}/jobs`

### Migrations

- `006_order_jobs.sql`

### Breaking / QA impact

- Do not rely on GET side effects for status; poll after delay (Compose demo: `30s`)
- Cover async transitions, restart with pending job, forbidden manual jumps

---

## Release 0.8.0 — 2026-08-29

**Theme:** Delivery + checkout

### Added / Changed

- User addresses CRUD; pickup points seed (5 active + 1 inactive)
- `CreateOrder` requires `deliveryMethod` + `addressId` or `pickupPointId`
- Delivery fee: courier **29900** (free if cart subtotal ≥ **500000**); pickup **0**
- Order stores delivery snapshot + fee; UI checkout delivery step before card pay
- Orders UI shows delivery method / address / fee

### Migrations

- `005_delivery.sql`

### Breaking / QA impact

- `POST /v1/orders` without delivery fields → error
- Cover courier/pickup happy paths, fee threshold, inactive PVZ, address CRUD + snapshot

---

## Release 0.7.0 — 2026-08-16

**Theme:** Vue 3 frontend

### Added / Changed

- Replaced vanilla JS SPA with **Vue 3 + Vite + vue-router** (hash routes unchanged: `#/login`, `#/catalog`, `#/cart`)
- UI build output: `web/dist`; gateway default `WEB_DIR=web/dist`
- **Dockerfile** multi-stage: Node builds UI, then Go builds API; serves `web/dist`
- Same REST flows and `data-testid` locators for QA automation
- Local UI: `cd web && npm install && npm run build` (or `npm run dev` with API proxy)

### Migrations

- None

### Breaking / QA impact

- Docker rebuild required to pick up Vue build
- Locators preserved; smoke UI paths unchanged

---

## Release 0.6.0 — 2026-08-14

**Theme:** Web UI (login, catalog, cart)

### Added

- **Static SPA** under [`web/`](web/) — hash routes `#/login`, `#/catalog`, `#/cart`
- Login / register against REST; JWT session in `localStorage`
- Catalog with brand filters and add-to-cart
- Cart: remove/clear, promocode apply/clear, totals (subtotal/discount/total, combo flag, TTL), checkout via `POST /v1/orders`
- HTTP gateway serves UI from `WEB_DIR` (default `web`); API remains under `/v1/`
- CORS on the HTTP gateway for browser clients
- `data-testid` attributes for UI automation

### Changed

- **Dockerfile** copies `web/` and sets `WEB_DIR=/app/web`
- [`README.md`](README.md) documents UI at `http://localhost:8080/`

### Migrations

- None

### Breaking / QA impact

- Open `http://localhost:8080/` for UI smoke; REST contract unchanged
- Cover login/register, catalog filter + add, cart promo + checkout paths in UI tests

---

## Release 0.5.1 — 2026-08-05

**Theme:** Hard delete order + Docker protobuf codegen

### Added

- **`DELETE /v1/orders/{order_id}`** (`OrderService.DeleteOrder`) — hard delete (order + items) for **any** status; JWT required; owner or admin
- Distinct from soft cancel (`POST /v1/orders/{order_id}/cancel`, only from `CREATED`)

### Changed

- **Dockerfile** installs `protoc` (Alpine `protobuf` + `protobuf-dev` for well-known types), `make`, and Go plugins (`protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-grpc-gateway`); runs `make generate` before `go build`
- [`Docs.MD`](Docs.MD) and [`README.md`](README.md) document DeleteOrder and Docker codegen

### Migrations

- None

### Breaking / QA impact

- New protected endpoint: `DELETE /v1/orders/{order_id}`
- Cover delete for `CREATED` / `PAID` / `SHIPPED` / `CANCELLED` / `COMPLETED`; wrong owner → 403; missing → 404
- Docker rebuild no longer depends on a pre-generated local `gen/` tree

---

## Release 0.5.0 — 2026-08-05

**Theme:** Delete user account

### Added

- **`DELETE /v1/users/{user_id}`** (`UserService.DeleteUser`) — JWT required; path `user_id` must match token `sub`
- Deletion allowed when the user has no orders, or only terminal orders (`COMPLETED` / `CANCELLED`); those orders (and items) are removed with the user
- Blocked with **FailedPrecondition** if any active order remains (`CREATED` / `PAID` / `SHIPPED`)

### Changed

- Auth interceptor requires JWT for `DeleteUser` only; Register / Login / GetUser stay public
- [`Docs.MD`](Docs.MD) and [`README.md`](README.md) document the new endpoint and rules

### Migrations

- None (uses existing `orders` FK `ON DELETE RESTRICT`; terminal orders are deleted in-app before the user row)

### Breaking / QA impact

- New protected endpoint for regression: `DELETE /v1/users/{user_id}`
- Cover cases: no orders → 200; only completed/cancelled → 200 + orders gone; active order → FailedPrecondition; wrong/missing token → 401/403

---

## Release 0.4.0 — 2026-07-26

**Theme:** Cart pricing, order lifecycle, admin promocodes

### Added

- **Promocodes (user):** `POST/DELETE /v1/users/{user_id}/cart/promocode`
- **Cart pricing response:** `subtotalCents`, `discountCents`, `appliedPromocode`, `comboDiscountApplied`, `expiresAt`
- **Combo rule:** NVIDIA + iPhone → 10% (see Docs.MD for expected rules)
- **Cart TTL:** 30 minutes inactivity
- **Order status:** `CancelOrder`, `UpdateOrderStatus` (`CREATED` → `PAID` / `CANCELLED`)
- **Auto progression:** `PAID` → `SHIPPED` (10m) → `COMPLETED` (10m), evaluated on `GET` order
- **Status** `ORDER_STATUS_COMPLETED`
- **Admin promocodes:** CRUD under `/v1/admin/promocodes` (`PromoService`)
- **Product `brand`:** `apple` / `samsung` / `nvidia` / `amd`
- **User `role`:** `user` \| `admin` in JWT; seed admin `admin@store.local` / `admin123`
- Seed promocodes: `SAVE10`, `FLAT500`, `WELCOME`

### Changed

- Cart & Order remain JWT-protected; admin routes require `role=admin`
- Order `GET` may advance status by timers
- [`Docs.MD`](Docs.MD) updated for mentees (auth, pricing, statuses, admin)

### Migrations

- `migrations/004_cart_pricing.sql` — `products.brand`, `carts`, `promocodes`, `users.role`, `orders.updated_at`, admin + promo seeds

### Breaking / QA impact

- Fresh DB volume required if upgrading from 0.3.x (`docker compose down -v`)
- Catalog JSON now includes `brand`
- Cart JSON has extra pricing fields
- Order status enum includes `COMPLETED`
- New endpoints to cover in regression suites (promo, cancel, status, admin)

---

## Release 0.3.0 — ~2026-07

**Theme:** Users & JWT auth

### Added

- `UserService`: register, login, get user
- JWT (`Authorization: Bearer`) for Cart and Order
- `user_id` must match token subject (403 otherwise)
- Migration `003_users.sql` — `users` table; cart/orders `user_id` as UUID FK

### Changed

- Cart/Order no longer accept anonymous string ids like `user-1`
- Flow: register/login → token → cart → order

### Breaking / QA impact

- All cart/order calls need a real registered user UUID + JWT
- Wipe DB if old text `user_id` values remain

---

## Release 0.2.0 — ~2026-06/07

**Theme:** PostgreSQL persistence

### Added

- Postgres via Docker Compose
- Migrations `001_init.sql`, `002_seed.sql` (50 products)
- Persistence for products, cart_items, orders / order_items

### Changed

- In-memory store replaced with SQL repositories
- Seed catalog stable UUIDs for docs and tests

### Breaking / QA impact

- Requires Docker Postgres (or local DSN)
- Data survives restarts (until volume wipe)

---

## Release 0.1.0 — init

**Theme:** First Store API simulator

### Added

- gRPC + HTTP gateway (`:50051` / `:8080`)
- Catalog: list / get product
- Cart: add / remove / get / clear
- Order: create from cart / get by id
- [`Docs.MD`](Docs.MD) API documentation for mentees

---

## How to add the next release

1. Bump section at the **top** (`Release x.y.z — YYYY-MM-DD`).
2. Fill **Added / Changed / Migrations / Breaking**.
3. Update [`Docs.MD`](Docs.MD) for mentees (expected behavior only).
4. Mentors: keep intentional-bug notes in gitignored `docs/BUGS.md` if needed.
