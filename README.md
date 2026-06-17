# Chirpy

A Twitter-like social media backend built in Go as part of the Boot.dev Backend Development course.

The goal of this project is to learn backend engineering concepts by building a real web application from scratch using:

- Go
- PostgreSQL
- SQLC
- Goose
- REST APIs
- HTTP Servers
- JSON APIs
- Database Migrations

---

# Features

## Static Website

Serves a static frontend from:

```
/app/
```

using Go's built-in file server.

---

## Health Check Endpoint

```
GET /api/healthz
```

Returns:

```json
OK
```

Used by monitoring systems and load balancers to determine if the application is healthy.

---

## User Creation

```
POST /api/users
```

Request:

```json
{
  "email": "user@example.com"
}
```

Response:

```json
{
  "id": "uuid",
  "created_at": "timestamp",
  "updated_at": "timestamp",
  "email": "user@example.com"
}
```

Creates a user in PostgreSQL.

---

## Chirp Creation

```
POST /api/chirps
```

Request:

```json
{
  "body": "Hello World",
  "user_id": "uuid"
}
```

Response:

```json
{
  "id": "uuid",
  "created_at": "timestamp",
  "updated_at": "timestamp",
  "body": "Hello World",
  "user_id": "uuid"
}
```

Creates a chirp associated with a user.

---

## Chirp Validation

Chirps:

- Must be 140 characters or less
- Cannot contain banned words

Banned words:

- kerfuffle
- sharbert
- fornax

Example:

Input:

```text
I love kerfuffle
```

Output:

```text
I love ****
```

---

## Admin Metrics

```
GET /admin/metrics
```

Returns an HTML page displaying the number of site visits.

Example:

```html
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited 10 times!</p>
  </body>
</html>
```

---

## Admin Reset

```
POST /admin/reset
```

Resets:

- Website visit counter
- All users in the database

Protected by:

```env
PLATFORM=dev
```

Only available in development environments.

---

# Technologies Used

## Go

Primary programming language.

Used for:

- HTTP server
- JSON APIs
- Middleware
- Business logic

---

## PostgreSQL

Relational database used for storing:

- Users
- Chirps

---

## Goose

Database migration tool.

Used to:

- Create tables
- Modify schema
- Roll back changes

Example:

```bash
goose postgres "$DB_URL" up
```

---

## SQLC

Generates type-safe Go code from SQL queries.

Benefits:

- No ORM
- Compile-time safety
- Better performance
- Full SQL control

Example:

```sql
-- name: CreateUser :one
INSERT INTO users ...
```

SQLC generates:

```go
dbQueries.CreateUser(...)
```

---

# Project Structure

```text
chirpy/
│
├── main.go
├── go.mod
├── .env
├── sqlc.yaml
│
├── internal/
│   └── database/
│       ├── db.go
│       ├── models.go
│       ├── users.sql.go
│       └── chirps.sql.go
│
├── sql/
│   ├── schema/
│   │   ├── 001_users.sql
│   │   ├── 002_users_uuid.sql
│   │   └── 003_chirps.sql
│   │
│   └── queries/
│       ├── users.sql
│       └── chirps.sql
│
└── index.html
```

---

# Database Schema

## Users

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    email TEXT NOT NULL UNIQUE
);
```

---

## Chirps

```sql
CREATE TABLE chirps (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    body TEXT NOT NULL,
    user_id UUID NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE
);
```

---

# Environment Variables

Create a `.env` file:

```env
DB_URL=postgres://postgres:postgres@localhost:5432/chirpy?sslmode=disable
PLATFORM=dev
```

---

# Installation

## Clone

```bash
git clone <repository-url>
cd chirpy
```

---

## Install Dependencies

```bash
go mod tidy
```

---

## Install Goose

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

---

## Install SQLC

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

---

# Database Setup

Create database:

```sql
CREATE DATABASE chirpy;
```

Run migrations:

```bash
goose -dir sql/schema postgres "$DB_URL" up
```

Generate SQLC code:

```bash
sqlc generate
```

---

# Running The Server

```bash
go build -o chirpy
./chirpy
```

Server runs on:

```
http://localhost:8080
```

---

# Example Requests

Create User:

```bash
curl -X POST localhost:8080/api/users \
-H "Content-Type: application/json" \
-d '{"email":"user@example.com"}'
```

Create Chirp:

```bash
curl -X POST localhost:8080/api/chirps \
-H "Content-Type: application/json" \
-d '{
  "body":"hello world",
  "user_id":"USER_UUID"
}'
```

Health Check:

```bash
curl localhost:8080/api/healthz
```

---

# Concepts Learned

This project demonstrates:

- HTTP servers in Go
- Routing with ServeMux
- Middleware
- JSON encoding/decoding
- REST API design
- PostgreSQL
- Database migrations
- SQLC code generation
- Environment variables
- Atomic counters
- Struct methods
- Dependency injection
- Error handling
- Context usage
- UUIDs
- Database relationships
- Foreign keys
- Cascading deletes

---

# Future Improvements

- Authentication
- Password hashing
- JWT tokens
- User login
- User profiles
- Chirp feeds
- Pagination
- Likes and comments
- Following system
- Production deployment