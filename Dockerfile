# --- Build stage ---
FROM golang:1.23-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux go build -o /out/migrate ./cmd/migrate

# --- Runtime stage ---
FROM alpine:3.20

RUN adduser -D -u 10001 casemind && apk add --no-cache ca-certificates curl
USER casemind
WORKDIR /app
COPY --from=build /out/api /app/api
COPY --from=build /out/migrate /app/migrate

EXPOSE 8080
ENTRYPOINT ["/app/api"]
