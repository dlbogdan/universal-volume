FROM ubuntu:22.04 AS base

LABEL org.opencontainers.image.authors="bogdan.dumitru@me.com"

ENV GO_VERSION=1.20

RUN apt-get update && apt-get install -yq software-properties-common wget

FROM base AS go-builder

RUN add-apt-repository -y ppa:longsleep/golang-backports \
    && apt update \
    && apt install golang-$GO_VERSION-go -yq
    #    git golang-$GO_VERSION-go

ENV GOPATH=/go
ENV PATH=$GOPATH/bin:/usr/local/go/bin:$PATH
ENV PATH=/usr/lib/go-$GO_VERSION/bin:$PATH
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

WORKDIR /go/src
COPY go.mod go.sum ./
RUN go mod tidy \ 
    && go mod download 

COPY go.* main.go ./


RUN set -ex  \
 && go install


FROM base

RUN apt-get update && apt-get install -yq \
        kmod vim \
    && mkdir -p /run/docker/plugins


COPY --from=go-builder /go/bin/docker-plugin-volume-test .
CMD ["docker-plugin-volume-test"]
