FROM golang:1.25.7-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /src

COPY . .

RUN go mod download

ENV CGO_ENABLED=0 GOOS=linux GOARCH=arm64
RUN go build -o /out/storm ./cmd/storm

FROM alpine:3.20

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

RUN chown root:appgroup /app && chmod 2775 /app

COPY --chown=appuser:appgroup --from=builder /out/storm /app/storm
COPY --chown=appuser:appgroup --from=builder /src/migrations /app/migrations

USER appuser

EXPOSE 8080
ENTRYPOINT ["/app/api"]