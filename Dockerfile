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
COPY --from=backend-builder /app/internal/adapter/out/persistence/sqlite/migration_files ./internal/adapter/out/persistence/sqlite/migration_files
COPY --from=frontend-builder /app/frontend/dist ./web
EXPOSE 8080 8081
CMD ["./kanbanai", "serve"]
