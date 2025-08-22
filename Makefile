up:
	docker compose --env-file config/.env up

upd:
	docker compose --env-file config/.env up -d

updb:
	docker compose --env-file config/.env up -d --build

down:
	docker compose --env-file config/.env down

test_orders:
	cd order_service && \
		go test ./tests -v --coverprofile=./tests/cover.out --coverpkg=./pkg/pkgports/adapters/cache/lru && \
		go tool cover --html=./tests/cover.out -o ./tests/cover.html

lint_orders:
	cd order_service && \
		golint ./... && \
		golangci-lint run ./...

test:
	cd integration_tests && \
		docker compose --env-file ./.env up -d order_service simulator_service nginx && \
		docker compose --env-file ./.env up --build e2e_test
	cd integration_tests && \
		docker compose --env-file ./.env down
