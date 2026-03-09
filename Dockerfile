FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /fhir-goals-engine ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o /seed seed.go

FROM alpine:3.19
RUN apk --no-cache add ca-certificates curl bash python3 postgresql-client
WORKDIR /app
COPY --from=builder /fhir-goals-engine .
COPY --from=builder /seed .
COPY static/ ./static/
COPY migrations/ ./migrations/
COPY postman/ ./postman/
EXPOSE 8080
CMD ["/bin/sh", "-c", "psql \"$DATABASE_URL\" -f migrations/001_create_tables.up.sql && ./seed && ./fhir-goals-engine"]
