APP_NAME   := hserv
CMD_DIR    := ./cmd/hserv

.PHONY: build run test clean docker-build docker-push

build:
	go build -o bin/$(APP_NAME) $(CMD_DIR)

run:
	go run $(CMD_DIR)

test:
	go test ./...

clean:
	rm -rf bin

docker-build:
	TAG="$${TAG:-latest}"; \
	docker build -t "$(APP_NAME):$$TAG" -f build/Dockerfile .

docker-push: docker-build
	@if [ -z "$$DOCKER_USER" ]; then echo "DOCKER_USER environment variable not set"; exit 1; fi
	TAG="$${TAG:-latest}"; \
	IMAGE_NAME_LOCAL="$(APP_NAME):$$TAG"; \
	IMAGE_NAME_REMOTE="$$DOCKER_USER/$(APP_NAME):$$TAG"; \
	docker tag "$$IMAGE_NAME_LOCAL" "$$IMAGE_NAME_REMOTE"; \
	docker push "$$IMAGE_NAME_REMOTE"
