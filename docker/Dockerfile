FROM golang:1.24-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /bin/myconfig .

FROM ubuntu:24.04

RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /bin/myconfig /usr/local/bin/myconfig

WORKDIR /home/ubuntu
COPY --chown=ubuntu:ubuntu docker/tests /home/ubuntu/tests
RUN chmod +x /home/ubuntu/tests/*.sh

USER ubuntu
ENV HOME=/home/ubuntu

ENTRYPOINT ["/home/ubuntu/tests/run-all.sh"]

