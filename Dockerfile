FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /fhir-goals-engine ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates curl bash python3
WORKDIR /app
COPY --from=builder /fhir-goals-engine .
COPY static/ ./static/
COPY migrations/ ./migrations/
COPY postman/ ./postman/
EXPOSE 8080
CMD ["./fhir-goals-engine"]
