
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY ../go.mod ../go.sum ./

RUN go mod download

COPY ../ ./

ARG APP_PATH=performance/
WORKDIR /app/${APP_PATH}
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/bin/app .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/bin/app .

EXPOSE 8080

CMD ["./app"]