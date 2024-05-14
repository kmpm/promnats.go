
FROM golang:1.21 as builder

RUN useradd -u 10001 promnats

WORKDIR /build/
 
ARG APPNAME=promnats
ARG VERSION=0.0.0-dev   
# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

COPY . /build/
RUN go build -ldflags "-s -w -X 'main.appVersion=${VERSION}'" -o ${APPNAME} ./cmd/promnats/


# running container
FROM busybox AS package
LABEL org.opencontainers.image.base.name="busybox"
LABEL org.opencontainers.image.source https://github.com/kmpm/promnats.go

ARG USER=default

WORKDIR /

ARG APPNAME=promnats
ARG VERSION=0.0.0-dev

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /build/${APPNAME} .
# COPY entrypoint.sh /usr/local/bin/

USER promnats 

EXPOSE 8083

ENTRYPOINT ["/promnats"]

