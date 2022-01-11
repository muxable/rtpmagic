# syntax=docker/dockerfile:1

FROM golang:1.17-alpine


WORKDIR /app

COPY go.* ./

RUN go mod download

COPY . ./

RUN go build -v -o /server cmd/server/main.go

EXPOSE 5000/udp

CMD [ "/server" ]