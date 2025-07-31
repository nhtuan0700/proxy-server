.PHONY: run
run:
	docker compose -f ./deployments/docker-compose.yml up -d --build
