FROM golang:1.24 

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

# Build argument to choose which app to build
ARG APP_PATH=performance
RUN CGO_ENABLED=0 GOOS=linux go build -a -o app ./${APP_PATH}

EXPOSE 8080

CMD ["./app"]