FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o user-search-service .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/user-search-service .

RUN adduser -D appuser
USER appuser

EXPOSE 4005

CMD ["./user-search-service"] 