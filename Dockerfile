FROM golang:alpine as builder
RUN apk update && apk add --no-cache git

ADD ./ /src
WORKDIR /src

ARG TARGETOS
ARG TARGETARCH
ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH

RUN sh build.sh

FROM  us-docker.pkg.dev/appintegration-toolkit/images/integrationcli:latest as integration

FROM  alpine:3
RUN apk add --no-cache bash
RUN apk add sed

COPY integrations /integrations
COPY LICENSE /
WORKDIR /
COPY --from=integration /usr/local/bin/integrationcli /usr/local/bin
COPY --from=builder /src/bin/* /usr/local/bin/
ENTRYPOINT [ "workday-entity-export" ]