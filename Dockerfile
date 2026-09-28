FROM golang:1.27.1-alpine3.24 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o pingo main.go


FROM alpine:3.23 AS prod

WORKDIR /

COPY --from=build /app/pingo .

CMD ["./pingo"]
