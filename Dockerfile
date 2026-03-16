FROM golang:1.24 as builder
WORKDIR /pbac
# Copy local code to the container image.
COPY  .  ./


RUN go mod download


RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -v -o server ./cmd/server/main.go

FROM ubuntu:22.04

RUN apt-get update && \
    groupadd web && \
    useradd -ms /bin/bash webuser -g web && \
    mkdir -p /home/webuser; chown -R webuser.web /home/webuser && \
    mkdir -p /var/folders; chown -R webuser.web /var/folders && \
    apt-get -yq --no-install-recommends install \
    language-pack-en-base \
    software-properties-common && \
    locale-gen en_US.UTF-8 && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

ENV LANG en_US.UTF-8
ENV LANGUAGE en_US:en
ENV LC_ALL en_US.UTF-8

USER webuser
WORKDIR /home/webuser
# Copy the binary to the production image from the builder stage.
COPY --from=builder --chown=root:root --chmod=755  /pbac/server /home/webuser/server
COPY --from=builder --chown=root:root --chmod=755  /pbac/migrations /home/webuser/migrations


# Run the web service on container startup.

CMD ["/home/webuser/server"]
