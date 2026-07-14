# Installation Guide

This documentation describes the environment variables and dependencies required to operate the library system.

## Dependencies

- **Go:** >= 1.26.4
- **PostgreSQL:** >= 15
- **Node.js:** >= 20 (for the Svelte 5 frontend build process)

## Environment Variables (ENVs)

The application is configured strictly via environment variables. For local operation, a `.env` file can be created in the main directory.

| Variable | Data Type | Description |
|---|---|---|
| `PORT` | Integer / String | Defines the port on which the HTTP server listens (e.g., `8081`). |
| `COOKIE_SECURE` | Boolean | Controls the `Secure` flag of HTTP cookies (`true` in production for HTTPS). |
| `DATABASE_URL` | String | Complete PostgreSQL Connection String (e.g., `postgres://user:pass@host:port/dbname`). |
| `JWT_SECRET` | String | Symmetric cryptographic key for JSON Web Token signature (minimum 32 characters). |
| `INITIAL_ADMIN_EMAIL` | String | Email address for the primary system administrator (only relevant for initial bootstrapping of an empty database). |
| `INITIAL_ADMIN_PASSWORD` | String | Plaintext password for the primary system administrator (will be cryptographically hashed upon creation). |
