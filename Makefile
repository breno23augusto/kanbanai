.PHONY: build run test lint migrate dev clean

build:
	go build -o bin/kanbanai cmd/kanbanai/main.go

run: build
	./bin/kanbanai serve

dev:
	go run cmd/kanbanai/main.go serve

test:
	go test ./... -v -race -cover

vet:
	go vet ./...

lint:
	golangci-lint run ./...

migrate:
	go run cmd/kanbanai/main.go migrate

clean:
	rm -rf bin/ data/

frontend-dev:
	cd frontend && npm run dev

frontend-build:
	cd frontend && npm run build
	rm -rf web && mkdir -p web && cp -r frontend/dist/* web/

docker-build:
	docker build -t kanbanai .

docker-run:
	docker run -p 8080:8080 -p 8081:8081 kanbanai
