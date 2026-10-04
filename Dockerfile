# golang:1.27.1-trixie
# https://hub.docker.com/layers/library/golang/1.27.1-trixie/images/sha256-fbcb99f2c5f6572a8738b97b614b12f7dfeda28a64195ee65bec0f8fd224b574
FROM golang@sha256:3b77fc618ec235a1ab412de7737f120dd507c57e8d87de4cbb7994fb94275ed5 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go tool task build


# alpine:3.24.2
# https://hub.docker.com/layers/library/alpine/3.24.2/images/sha256-c54a80678a9e7a744b39d478329d5eb8e568c2e0321defd14cab57047557e202
FROM alpine@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS prod

WORKDIR /

COPY --from=build /app/pingo .

CMD ["./pingo"]
