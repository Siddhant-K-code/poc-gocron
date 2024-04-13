FROM golang:latest as builder

WORKDIR /app

COPY go.mod go.sum ./
COPY main.go ./

RUN go mod download
RUN go build -o main .

FROM debian:latest

WORKDIR /app

COPY --from=builder /app/main ./

RUN apt-get update && apt-get install -y postgresql-client default-mysql-client ca-certificates

CMD ["./main"]
