# KanbanAI — Operações

## 1. Makefile

```makefile
.PHONY: build run test lint migrate dev

build:
	go build -o bin/kanbanai cmd/kanbanai/main.go

run: build
	./bin/kanbanai serve

dev:
	go run cmd/kanbanai/main.go serve

test:
	go test ./... -v -race -cover

lint:
	golangci-lint run ./...

migrate:
	go run cmd/kanbanai/main.go migrate

frontend-dev:
	cd frontend && npm run dev

frontend-build:
	cd frontend && npm run build
```

---

## 2. Dockerfile

Multi-stage build para empacotar backend Go + frontend React:

```dockerfile
# Stage 1: Build Go Backend
FROM golang:1.26-alpine AS backend-builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o kanbanai cmd/kanbanai/main.go

# Stage 2: Build Frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# Stage 3: Final Image
FROM alpine:latest
RUN apk add --no-cache sqlite ca-certificates
WORKDIR /app
COPY --from=backend-builder /app/kanbanai .
COPY --from=frontend-builder /app/frontend/dist ./web
EXPOSE 8080 8081
CMD ["./kanbanai", "serve"]
```

---

## 3. Graceful Shutdown

O KanbanAI reage a sinais de terminação do SO (`SIGINT`, `SIGTERM`):

1. **Interrupção de Novos Clientes**: O servidor HTTP/API para de receber novos requests.
2. **Drenagem de Eventos**: O SSE Broker aguarda até 5 segundos para transmitir mensagens na fila e encerra conexões ativas.
3. **Cancelamento de Processos Ativos**: O context de aplicação dos harnesses é cancelado, terminando processos CLI em andamento.
4. **Fechamento do DB**: A conexão com o banco SQLite é devidamente fechada.

---

## 4. Performance — Boas Práticas Go

- **Goroutines**: O `PhaseOrchestrator` executa harness dispatch em goroutines separadas.
- **Channels**: SSE Broker usa channels para comunicação non-blocking entre goroutines.
- **sync.Pool**: Reutilização de buffers no formatter SSE.
- **Context propagation**: Todos os métodos recebem `context.Context` para cancelamento e timeouts.
- **Prepared statements**: SQLite queries usam prepared statements cacheados.
- **Batch inserts**: Event logs podem ser inseridos em batch quando há muitos eventos.
- **Read-Write Mutex**: `sync.RWMutex` em caches e maps compartilhados.
- **Struct embedding**: Uso de composição ao invés de herança para reutilização.
