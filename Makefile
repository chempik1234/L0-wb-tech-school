up:
	docker compose --env-file config/.env up

upd:
	docker compose --env-file config/.env up -d

updb:
	docker compose --env-file config/.env up -d --build

down:
	docker compose --env-file config/.env down