# Chirpy API (Go, PostgreSQL) –

**Chirpy** is a mini–Twitter clone built in Go. It provides a RESTful JSON API for user accounts, “chirps” (short text posts up to 140 characters), and a premium subscription (“Chirpy Red”) via a webhook. Under the hood it uses PostgreSQL (with [sqlc](https://sqlc.dev/) for type-safe SQL queries), [goose](https://github.com/pressly/goose) for schema migrations, JWT tokens for authentication, and Argon2id for password hashing. Chirps are automatically filtered for profanity, and only the author may delete a chirp. Administrators (in a “dev” platform mode) can reset the database and view simple metrics. The service also serves static files under `/app/`.  

The high-level architecture is a single Go HTTP server behind a standard HTTP router.  Clients (e.g. a web or mobile UI) make requests to endpoints like `POST /api/users` or `GET /api/chirps`.  The server decodes requests, authenticates users via JWT, runs SQL queries against PostgreSQL, and returns JSON responses.  Environment variables (in a `.env` file) configure the service, including the database URL and secret keys.  

For example, Chirpy supports: user registration/login, JWT-based auth with refresh tokens, profile updates, creating/listing/deleting chirps (with profanity filtering), and a webhook endpoint to upgrade users to “Chirpy Red” premium membership.  The service is instrumented with readiness (`GET /api/healthz`) and basic admin endpoints (`GET /admin/metrics`, `POST /admin/reset` in dev mode).  

## Setup & Local Run

1. **Prerequisites**: Install [Go (v1.24+)](https://go.dev/doc/install), [PostgreSQL](https://www.postgresql.org/download/), and command-line tools [goose](https://github.com/pressly/goose) and [sqlc](https://sqlc.dev/). Also install [`github.com/joho/godotenv`](https://github.com/joho/godotenv) for loading `.env` (optional but recommended). Finally, get the required Go dependencies:

   ```bash
   go install github.com/pressly/goose/v3/cmd/goose@latest
   go install github.com/kyleconroy/sqlc/cmd/sqlc@latest
   go get github.com/joho/godotenv
   ```

2. **Clone and build**:
   ```bash
   git clone <repo-url> Chirpy
   cd Chirpy
   go mod download
   ```

3. **Environment variables**: Create a `.env` file in the project root with **at least** the following variables (see [gr​a​in​me/Chirpy README][64] for an example):  

   ```dotenv
   DB_URL=postgres://user:password@localhost:5432/chirpydb?sslmode=disable
   JWT_SECRET=your-jwt-secret-key
   PLATFORM=dev
   POLKA_KEY=your-polka-api-key
   ``` 

   - `DB_URL`: PostgreSQL connection string (with `?sslmode=disable` in dev).  
   - `JWT_SECRET`: a long random string used to sign JWTs.  
   - `PLATFORM`: set to `dev` for development (enables the `/admin/reset` endpoint); use `prod` in production.  
   - `POLKA_KEY`: the secret API key expected on the Polka webhook (keep this truly secret).  

   *(Note: The code uses `os.Getenv("SECRET")` or `os.Getenv("JWT_SECRET")` for the JWT key depending on version; make sure it matches your code.)*  

4. **Database setup**: Create the `chirpydb` database in Postgres.  Run migrations using goose, for example:  
   ```bash
   goose -dir sql/schema postgres "$DB_URL" up
   ```  
   This applies all pending schema migrations (goose will create a `schema_migrations` table).  You can also step through, rollback, or check status with goose [as documented][51].  

   A sample migration to add the “chirpy_red” column might look like:  
   ```sql
   -- +goose Up
   ALTER TABLE users ADD COLUMN is_chirpy_red BOOLEAN NOT NULL DEFAULT FALSE;
   -- +goose Down
   ALTER TABLE users DROP COLUMN is_chirpy_red;
   ```  
   (Each goose migration file in `sql/schema/` should have `-- +goose Up` and `-- +goose Down` sections.)  For details on goose commands, see the goose documentation or examples.

5. **SQLC codegen**: The `.sql` query files are in `sql/queries/`.  Whenever you edit SQL queries, run `sqlc generate` to re-generate Go query code in `internal/database`.  For example:  
   ```bash
   sqlc generate
   ```  
   SQLC turns your SQL (marked with `-- name: <QueryName> :one|:many|:exec`) into Go methods.  Make sure each query has a proper name and correct return modifier (e.g. `:one` for single-row results, `:many` for multiple rows, `:exec` for no-return DML).  The generated code will expect the standard Postgres driver (`github.com/lib/pq`) and [google/uuid][23] if you use UUIDs in queries.

6. **Run the server**: Start the API server on `:8080` by:  
   ```bash
   go run main.go
   ```  
   If all is well, you’ll see "Starting server on :8080".  The health check endpoint `GET /api/healthz` will then respond with `200 OK`.  

7. **Boot.dev CLI tests (optional)**: If you’re following the Boot.dev curriculum, install the [Boot.dev CLI](https://github.com/bootdotdev/bootdev) and run:  
   ```bash
   bootdev login         # once, to authenticate
   bootdev run <test-id> # to run tests (debug mode)
   bootdev run <test-id> -s  # to submit/check
   ```  
   This will execute predefined HTTP requests against your server.  Use `run` to debug errors; the `-s` flag will report pass/fail.

## Architecture & Code Structure

Chirpy is structured in packages:

- **`main.go`**: Loads environment (via `godotenv`), reads `DB_URL`, `JWT_SECRET`, `PLATFORM`, `POLKA_KEY` into a configuration struct.  It opens the Postgres connection (`sql.Open("postgres", dbURL)`) and initializes the generated `database.Queries`.  It then sets up HTTP routes on `http.DefaultServeMux` and starts the server.  For example (excerpt):  

  ```go
  godotenv.Load()
  cfg.Platform = os.Getenv("PLATFORM")
  cfg.Secret = os.Getenv("JWT_SECRET")       // JWT signing key
  cfg.PolkaKey = os.Getenv("POLKA_KEY")     // Polka webhook key
  cfg.DbURL = os.Getenv("DB_URL")
  db, err := sql.Open("postgres", cfg.DbURL)
  cfg.DbQueries = database.New(db)
  mux := http.NewServeMux()
  RegisterEndpoints(mux, &cfg)
  log.Println("Starting server on :8080")
  http.ListenAndServe(":8080", mux)
  ```  

- **`internal/auth/`**: Authentication helpers. This contains functions to hash and check passwords (using [Argon2id][73] for strong hashing) and to create/validate JWTs (using `github.com/golang-jwt/jwt/v5`).  It also has helper functions like `GetBearerToken(header http.Header)` to parse the `Authorization: Bearer <token>` header, and `GetAPIKey(header http.Header)` to parse `Authorization: ApiKey <key>`. JWTs here include a custom claim for the user ID and expire in (typically) 1 hour.  Upon each authenticated request, `ValidateJWT(token, secret)` checks the signature and expiry, returning the user’s UUID if valid.

- **`internal/database/`**: This wraps the code generated by sqlc. After running `sqlc generate`, you get a `Queries` struct with methods like `CreateUser(ctx, params)`, `GetUserByEmail`, `CreateChirp`, etc.  For example, `dbQueries.CreateUser(ctx, database.CreateUserParams{Email, HashedPassword})` will insert a new user and return a struct containing all columns (including the new UUID). Similarly, there are methods like `GetChirp`, `GetChirps`, `GetChirpsByAuthor`, `DeleteChirp`, `UpgradeUserToChirpyRed`, etc., corresponding to the SQL queries defined.  (SQLC ensures type safety – e.g. UUID columns become `uuid.UUID` in Go, and missing or extra columns are compile-time errors.)  

- **`sql/queries/`**: Contains `.sql` files with the raw SQL statements. Each query begins with a comment like `-- name: CreateUser :one` (naming the Go method). The `:one` means one row is expected; use `:many` for multi-row (e.g. `GetChirps` returns multiple), or `:exec` for commands without a result set.  Example (in `user.sql`):  
  ```sql
  -- name: CreateUser :one
  INSERT INTO users (email, hashed_password) 
  VALUES ($1, $2)
  RETURNING *;
  ```
  After editing these, run `sqlc generate`.  *(Common pitfall: forgetting `:one` or using the wrong return clause can cause mismatched results.)*  

- **Handlers/Endpoints**: Chirpy’s HTTP API routes are registered in `main.go` or an `api` package.  Key endpoints include:  
  - **Users**  
    - `POST /api/users` – Create a new user.  Request JSON: `{"email": "...", "password": "..."}`.  Response: `201 Created` with JSON `{"id": <uuid>, "email": "...", "is_chirpy_red": false, "created_at": "...", "updated_at": "..."}`.  (The `is_chirpy_red` field is added via a migration; it starts as `false` by default.)  
    - `PUT /api/users` – Update current user’s email/password.  Requires `Authorization: Bearer <token>`.  Request JSON: `{"email": "...", "password": "..."}`.  Returns `200 OK` with the updated user JSON (including the new timestamps and `is_chirpy_red` flag).  

  - **Auth**  
    - `POST /api/login` – User login.  Request: `{"email": "...", "password": "..."}`.  On success, returns `200 OK` with JSON `{"id": <uuid>, "email": "...", "is_chirpy_red": <bool>, "token": "<jwt>", "refresh_token": "<token>"}`.  The `token` is a JWT (used in future `Bearer` headers), and `refresh_token` is a long random string stored server-side.  If credentials fail, returns `401 Unauthorized`.  
    - `POST /api/refresh` – Refresh JWT.  Requires header `Authorization: Bearer <refresh_token>`.  If the refresh token is valid (found in the database and not expired), returns `200 OK` with a new short-lived JWT: `{"token": "<new_jwt>"}`. If invalid, returns `401`.  
    - `POST /api/revoke` – Revoke refresh token.  Requires header `Authorization: Bearer <refresh_token>`.  This deletes the refresh token from the database, returning `204 No Content`.  

  - **Chirps (posts)**  
    - `POST /api/chirps` – Create a chirp (JWT required).  Request: `{"body":"Your message here"}`.  The body is auto-filtered for profanity (e.g. banned words like *kerfuffle*, *sharbert*, *fornax* are replaced with `****`) and must be ≤140 characters (else `400 Bad Request`).  On success, returns `201 Created` with the chirp JSON: `{"id": <uuid>, "body": "...", "user_id": <uuid>, "created_at": "...", "updated_at": "..."}`.  
    - `GET /api/chirps` – List chirps.  No auth required.  Supports optional query params `author_id=<user_uuid>` (to filter by user) and `sort=asc|desc` (to sort by creation time).  If `author_id` is given, the SQL query `GetChirpsByAuthor` is used to fetch only that user’s chirps.  Otherwise `GetChirps` returns all chirps.  The code then sorts them in memory by `CreatedAt` (ascending by default; descending if `sort=desc`).  (Filtering by author should be done in SQL for efficiency.)  Returns `200 OK` with JSON array of chirps (possibly empty).  *Example:*  
      ```bash
      curl "http://localhost:8080/api/chirps?author_id=<uuid>&sort=desc"
      ```  
    - `GET /api/chirps/{chirpID}` – Get one chirp by ID.  No auth.  Returns `200 OK` with the chirp JSON, or `404 Not Found` if the ID is invalid or missing.  
    - `DELETE /api/chirps/{chirpID}` – Delete a chirp.  Requires JWT.  First it looks up the chirp’s author from the DB; if the authenticated user ID doesn’t match, returns `403 Forbidden`.  Otherwise it deletes the chirp and returns `204 No Content`.  

  - **Webhooks (Polka)**  
    - `POST /api/polka/webhooks` – Handle payment webhooks from Polka.  This endpoint requires an `Authorization: ApiKey <key>` header.  If the key doesn’t match the `POLKA_KEY`, it returns `401 Unauthorized`.  The request body is JSON like `{"event":"user.upgraded","data":{"user_id":"<uuid>"}}`.  If `event` is anything other than `"user.upgraded"`, the handler simply returns `204 No Content` (ignore it).  If it *is* `"user.upgraded"`, it parses the `user_id` and calls the database query to set that user’s `is_chirpy_red = true`.  On success it returns `204 No Content`; if the user isn’t found, it returns `404 Not Found`.  This makes sure only Polka (who knows the API key) can upgrade users to premium..  

  - **Admin/Health**  
    - `GET /api/healthz` – Always returns `200 OK` with a plaintext “OK” (readiness check).  
    - `GET /admin/metrics` – (Dev) shows a simple HTML page with visit counters (hits on the static file server).  
    - `POST /admin/reset` – (Dev only, `PLATFORM=dev`) wipes all users (and cascading chirps) and resets visit counters.  Returns `200 OK`.  

Each handler carefully sets the HTTP status codes and JSON fields as above.  For example, creating a user returns JSON like:  

```http
POST /api/users
Content-Type: application/json

{"email":"walt@breakingbad.com","password":"123456"}
```
```http
HTTP/1.1 201 Created
Content-Type: application/json

{
  "id": "bbb39dc5-0ce4-4229-a8be-ddea5c7837a2",
  "email": "walt@breakingbad.com",
  "is_chirpy_red": false,
  "created_at": "2026-06-18T02:54:39Z",
  "updated_at": "2026-06-18T02:54:39Z"
}
```  
*(Fields `created_at`, `updated_at`, and the UUID are automatically set in DB.)*  Login returns the same fields plus `token` and `refresh_token`.  To delete a chirp:  

```bash
curl -X DELETE http://localhost:8080/api/chirps/2d19eaeb-1d5d-409c-9709-3ab48c26b7f4 \
     -H "Authorization: Bearer <your_jwt>"
```  
which yields `204 No Content`.  (The permissions check ensures only the chirp’s author can delete it.)

## Authentication & Security

Chirpy uses **JWT (JSON Web Tokens)** for stateless authentication.  Upon login, the server issues a signed JWT containing the user’s ID (and an expiration claim). The client must then include `Authorization: Bearer <token>` on protected requests.  The server calls `jwt.ParseWithClaims` to verify signature and expiry using the shared secret (`JWT_SECRET`). A valid token means the user is authenticated as that ID.  (Note: JWTs are signed but **not encrypted**; anyone can read their claims if they have the token.  Thus, we never put sensitive data in a JWT – only user ID and expiry are stored.)  

We also issue **refresh tokens**: long random strings stored (hashed) in the database with an expiry.  The client gets one upon login and must use `POST /api/refresh` with `Authorization: Bearer <refresh_token>` to obtain a new JWT when the old one expires.  This allows short-lived access tokens and the ability to revoke sessions.  (If the refresh token is missing or invalid, `401 Unauthorized` is returned.)  Revoking simply deletes the token (via `POST /api/revoke`).  

Passwords are hashed using **Argon2id** (via `github.com/alexedwards/argon2id`). This is a modern, slow, salted hashing algorithm (stronger than plain bcrypt). The code uses `argon2id.CreateHash(password, params)` to hash and `argon2id.ComparePasswordAndHash` to verify. Never store plain passwords.  

For the Polka webhook, we use a separate API key mechanism: each request must include `Authorization: ApiKey <POLKA_KEY>`.  The handler checks that header and rejects (`401`) if it’s wrong.  This prevents unauthorized callers from hitting the webhook (only Polka knows the key).  

SQL queries use parameter binding (via sqlc) so SQL injection is not a concern.  Chirp text is sanitized by a simple word-filter (replacing banned words with asterisks), preventing obvious profanity.  We also enforce chirp length ≤140 characters at the handler level.  

Common security notes: use HTTPS in production to protect tokens in transit; keep all secrets (JWT_SECRET, POLKA_KEY, DB_URL credentials) out of source control (via `.env` or env vars).  Don’t use debug or admin endpoints (like `/admin/reset`) in production.  For further details on JWT best practices, see Boot.dev’s JWT guide.

## Database Migrations (Goose)

Chirpy’s database schema starts with tables for `users`, `chirps`, and `refresh_tokens`. The `users` table has `id (UUID PK)`, `email`, `hashed_password`, timestamps, and (after migration) `is_chirpy_red BOOLEAN DEFAULT FALSE`.  `chirps` links to `users` via a foreign key with `ON DELETE CASCADE`, so deleting a user removes their chirps. The `refresh_tokens` table stores a token string, user_id (FK), and expiry.

We use Goose to version migrations.  Each migration in `sql/schema/` looks like:
```sql
-- +goose Up
ALTER TABLE users ADD COLUMN is_chirpy_red BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose Down
ALTER TABLE users DROP COLUMN is_chirpy_red;
```
To apply all migrations, run:
```bash
goose -dir sql/schema postgres "$DB_URL" up
``` 
This will create/maintain a `schema_migrations` table and run any new scripts.  You can also use `goose down`, `goose status`, etc.  (The [DZone Goose tutorial][51] has a good Makefile example.)  Keep migrations small and incremental.  

## SQLC Queries

Instead of writing raw SQL in code, we use [sqlc](https://sqlc.dev/) to generate Go methods from `.sql` files.  For example, in `sql/queries/user.sql` we might have:
```sql
-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;
```
After `sqlc generate`, we can call `dbQueries.GetUserByEmail(ctx, email)`, which returns a Go struct with typed fields.  The `:one` suffix indicates exactly one row is expected.  For multi-row queries (e.g. `GetChirps`), use `:many`.  For commands like `DELETE`, use `:exec`.  SQLC also supports named return values.  See the [Boot.dev SQLC lesson][23] for examples: it notes that SQLC “generates Go code from SQL queries, making working with raw SQL easy and type-safe”.  

Some tips:
- Prefix `sqlc generate` in your build or CI so code stays up to date.  
- Include `UUID` columns (as `uuid.UUID` in Go) by adding `go get github.com/google/uuid`.  
- If you change the schema, write a goose migration **and** update your SQL queries accordingly (e.g. add the new column to RETURNING *).  
- Don’t check `.env` into git; [23] suggests adding it to `.gitignore` and using `go get` for dependencies.  

## Authentication Flow (Detailed)

1. **Registration**: `POST /api/users` – user submits email & password. The handler hashes the password (`argon2id`) and runs `CreateUser` in SQLC. The DB returns the new user (with UUID). We respond with JSON containing `id`, `email`, `is_chirpy_red:false`, etc.  

2. **Login**: `POST /api/login` – user submits credentials. We look up the user by email (`GetUserByEmail`). If not found, we return `401 Unauthorized`. Otherwise we compare the hashed password (`argon2id.ComparePasswordAndHash`). If valid, we generate:  
   - A JWT signed with `JWT_SECRET`, containing the user’s ID. (This uses `jwt-go` under the hood.)  
   - A random **refresh token** (a long string, e.g. from `crypto/rand`), which we store in DB with expiration (e.g. now+60 days).  
   We then return `200 OK` with JSON: `{"id":..., "email":..., "is_chirpy_red":..., "token": "<jwt>", "refresh_token": "<token>"}`.  

3. **Authenticated Requests**: Subsequent requests must include `Authorization: Bearer <jwt>` (or `ApiKey` for Polka).  Each handler calls a function like `ValidateJWT` which checks the signature and expiry. If invalid or expired, we return `401`. If valid, we obtain the `user_id` from the JWT claims and proceed.  

4. **Refresh**: To extend a session, the client calls `POST /api/refresh` with `Authorization: Bearer <refresh_token>`.  The handler runs the SQLC query `GetUserFromRefreshToken(token)`. If a user is found and token is not expired, we issue a new JWT and return it.  If not, we `401 Unauthorized`.  

5. **Revoke**: To log out, `POST /api/revoke` with the refresh token header. The handler calls `RevokeRefreshToken(token)` (SQL `DELETE`). Returns `204 No Content`.  

6. **Webhook (Polka)**: As noted, `POST /api/polka/webhooks` expects header `ApiKey <POLKA_KEY>`.  On correct key, it parses JSON `{"event":"...","data":...}`. If event=`"user.upgraded"`, it extracts `data.user_id` (a UUID) and runs `UpgradeUserToChirpyRed(userID)` in SQLC (which does `UPDATE users SET is_chirpy_red = true`).  Then returns `204` if user existed or `404` if not. For any other event, it immediately returns `204` (no error).  

## Testing & Debugging

Chirpy includes automated CLI tests (if using Boot.dev).  You can run them with the Boot.dev CLI (`bootdev run <ID>`).  When a test fails, check the HTTP request/response it expects.  Common issues:  

- **401 Unauthorized**: This means a required token/header is missing or invalid.  E.g. ensure you use `Authorization: Bearer <JWT>` on protected routes, and `Authorization: ApiKey <key>` (note the word "ApiKey") on the Polka endpoint.  A frequent mistake is forgetting the “Bearer ” prefix or misspelling the header.  
- **403 Forbidden**: Occurs if a user tries to delete another’s chirp. This is correct behavior.  
- **404 Not Found**: Could be due to an incorrect path (e.g. missing trailing slash in registration), or an invalid UUID.  Check your route registrations.  For example, `mux.HandleFunc("/api/chirps/{chirpID}", ...)` will not match `"/api/chirps"` – those are separate registrations.  
- **405 Method Not Allowed**: Means the HTTP method isn’t handled.  Ensure your handlers switch on `r.Method` or you’ve registered the correct method (see “Routing” below).  

- **500 Internal Error**: Often a bug in server code. For instance, the SQLC query signature must match usage. If you get an “assignment mismatch” error (e.g. *2 variables but UpgradeUserToChirpyRed returns 1*), it means your code called a function with the wrong number of return values. Check the generated method signature (maybe you used `:one` vs `:exec` incorrectly). Also check any JSON decoding errors.  

Use tools like `curl` or Postman to replicate test requests.  For example, to debug a Polka webhook test:  

```bash
curl -v -X POST http://localhost:8080/api/polka/webhooks \
     -H "Authorization: ApiKey wrongkey" \
     -H "Content-Type: application/json" \
     -d '{"event":"user.upgraded","data":{"user_id":"<uuid>"}}'
```  

This should give `401` (as expected if the key is wrong).  

The Boot.dev CLI tests may show the expected status and actual status (as in the problem conversation above).  Compare those to pinpoint issues.  

## Routing Approaches

In Go’s `net/http`, you can register handlers in two ways: method-specific routes or a single route with method switching. For example, one design is to do:

```go
mux.HandleFunc("/api/users", func(w, r *http.Request) {
    switch r.Method {
    case http.MethodPost:
        createUserHandler(w,r)
    case http.MethodPut:
        updateUserHandler(w,r)
    default:
        http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
    }
})
```

Alternatively, you can (with Go 1.20+ or using gorilla/mux) register different handlers per method/path combination. e.g.:

```go
mux.HandleFunc("POST /api/users", createUserHandler)
mux.HandleFunc("PUT /api/users", updateUserHandler)
```

Both achieve similar results.  The switch approach keeps related logic together but can become verbose.  The method-specific registration can be cleaner but requires either a multiplexer that supports it (or nesting muxes).  Neither is “right” – it’s a design choice.  

## Deployment Notes

- **Environment**: In production, set `PLATFORM=prod` (so admin reset is disabled). Ensure `JWT_SECRET` and `POLKA_KEY` are securely provided (e.g. via container secrets or environment). Use a strong secret for `JWT_SECRET` (at least 256 bits).  
- **Port**: The server listens on `:8080` by default. You can change this in `main.go` or via an env var if needed.  
- **HTTPS**: Put Chirpy behind an HTTPS reverse proxy or load balancer in prod.  
- **Logging & Monitoring**: Currently Chirpy prints to stdout on start; you may wish to add structured logging or metrics in a real setup.  
- **Scaling**: The stateless Go server can be run behind a load balancer. PostgreSQL is a single point of truth (could use read replicas for scaling reads). Ensure proper indexing: the primary keys (user id, chirp id) are indexed by default. If you anticipate frequent filtering by `author_id`, ensure it’s indexed (Postgres creates an index on foreign-key by default, and you may add indexes on `created_at` if sorting often).  
- **CI/CD**: Integrate `goose` migrations into your deployment pipeline. For example, see [51†L438-L445] on automating `goose up` in CI/CD.  

## Design Considerations (Interview Topics)

- **Why Go?** Concurrency primitives, performance, static binaries, strong standard library.  
- **Why SQLC vs ORM?** SQLC gives type-safe, explicit SQL without an ORM’s abstraction. It avoids runtime query mistakes by generating code at compile time. It’s easy to see exactly what SQL runs.  
- **Why JWT & Refresh?** JWT allows stateless auth (no server-side session state). It scales well with microservices or multiple servers (no shared session store needed). Refresh tokens allow short-lived JWTs for security. Trade-off: JWTs can’t be easily revoked, hence we keep them short (1h) and use revokable refresh tokens. In interviews, note that JWT payloads shouldn’t contain secrets (they’re just identity claims).  
- **Routing Trade-offs**: The code used net/http's default mux. For path parameters (like `/api/chirps/{id}`), it uses `r.URL.PathValue("chirpID")` (Go 1.20+ supports `{name}` patterns) or regex. Discuss alternatives (gorilla/mux or manual regex switches).  
- **DB Filtering vs In-Memory**: The original assignment required that filtering by `author_id` happen in SQL. Indeed we use `WHERE user_id = $1` in the query (SQLC’s `GetChirpsByAuthor`). Sorting we do in Go using `sort.Slice`, but we could also let Postgres sort via `ORDER BY created_at`. For large data, prefer pushing work to the DB (indexes, LIMIT/OFFSET for pagination, etc.).  
- **Indexing**: The `id` columns are primary keys. The `chirps.user_id` FK is implicitly indexed. If sorting or filtering by date often, adding an index on `created_at` might help.  
- **Security hardening**: We hashed passwords and secret keys are env vars. For added security, rate limit login attempts to prevent brute force. Also verify JWT claims (we use standard `jwt.StandardClaims`).  
- **File Server**: The code also serves static files under `/app/` using `http.FileServer`. We wrap it to count hits (metrics). In production, static assets are often served by a CDN or separate server.  

**Sample Interview Q&A**:  
- *Q: How do you revoke a JWT before it expires?* – We implement refresh tokens stored in DB; we can revoke by deleting the refresh token. The JWT itself is short-lived (1h) and not stored server-side (so truly stateless). One could also keep a token blacklist for full control.  
- *Q: How would you add pagination to GET /api/chirps?* – We could extend the query with `LIMIT`/`OFFSET` parameters (or better, use “cursor” pagination). SQLC supports dynamic queries or named parameters for this. Then clients could pass `?limit=20&offset=40`.  
- *Q: Why use Argon2id for passwords?* – It’s a modern, memory-hard hashing designed to resist GPU cracking, and Argon2id variant provides resistance against timing attacks. Compared to bcrypt, Argon2id has more tuning parameters. (Cite [64†L293-L300] which mentions Argon2id.)  

## Troubleshooting & FAQ

- **“404 Not Found” on POST /api/users**: Ensure the path is correct. (No trailing slash or mismatched case.) Also check `HandleFunc` registrations: e.g. registering `"/api/users"` covers both POST and PUT in our switch.  
- **“assignment mismatch” compiler error**: Likely due to a SQLC query that doesn’t match your code. Maybe you wrote `-- name: X :one` but the query returns no rows, or vice versa. Check the SQLC-generated code or query annotation.  
- **Login always fails (401)**: Check that the email exists and password hashing matches. Also ensure the DB was migrated/seeded with that user. Debug by attempting a known `CreateUser` then login.  
- **“Token is expired”**: The JWT issued has a 1-hour expiry. After that, `/api/login` is still required (or use the refresh token endpoint).  
- **Polka webhook not working (401 instead of 204)**: The header must be exactly `Authorization: ApiKey <your-key>`. It’s case-sensitive on “ApiKey” in our code. Also ensure `POLKA_KEY` in `.env` matches what you use.  
- **SQL errors about uuid**: We use `uuid.UUID` fields. Make sure you imported `"github.com/google/uuid"` and that your queries use `uuid_generate_v4()` or `gen_random_uuid()` (you may need the `pgcrypto` extension enabled for `gen_random_uuid()` as noted in Boot.dev [23†L125-L134]). In migration SQL use `gen_random_uuid()` in column defaults and enable the extension if needed.  

If you run into any other issues, double-check environment names (`JWT_SECRET` vs `SECRET`), run `goose up`, and read server logs. The Boot.dev Discord/forums are also helpful for debugging questions.

## References & Further Reading

- Official Go [`net/http`](https://pkg.go.dev/net/http) package documentation.  
- [Boot.dev Go SQLC Lesson][23] – explains sqlc queries, `.env` usage, and scanning.  
- [Boot.dev JWT Blog][40] – covers JWT basics and flow.  
- [Boot.dev JWT Review][41] – security considerations for JWT (immutability, no encryption).  
- [Boot.dev CLI Docs][72] – using `bootdev run` and CLI instructions.  
- [SQLC official docs](https://sqlc.dev/) – for writing queries and regenerating code.  
- [Goose repository](https://github.com/pressly/goose) – for migration usage examples.  
- Chirpy example code repositories (e.g. [neeeb1/chirpy][17] and [grainme/Chirpy][64]) for reference API usage.  
- PostgreSQL docs on [UUID generation](https://www.postgresql.org/docs/current/uuid-ossp.html) and [indexing](https://www.postgresql.org/docs/current/indexes.html).  

*(This README is meant to be exhaustive. Happy coding, and good luck with interviews!)*  

**Sources:** Chirpy code and related tutorials.  


