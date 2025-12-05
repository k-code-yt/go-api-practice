# Build stage
FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

# Build argument to choose which app to build
ARG APP_PATH=kafka-chans
RUN CGO_ENABLED=1 GOOS=linux go build -a -o app ./${APP_PATH}

# Final stage
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /root/

COPY --from=builder /app/app .

EXPOSE 8080

CMD ["./app"]