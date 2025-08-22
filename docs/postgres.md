### Deployment

Postgres is used as a **single container** with a custom Dockerfile and an alpine-based image

### SQL Scripts

Postgres automatically creates 1 database and 1 user with permissions to that database.

Names are set in .env: `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`

### Migrations

`Go-migrate` is used as the migration tool, the tables are stored in a schema named `order_service`

### Tables

`orders` -> `deliveries`, `payments` and (`order_items` 1:M)