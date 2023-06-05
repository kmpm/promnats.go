
FROM golang:1.20-alpine as builder
ARG GITHUB_USER=someuser
ARG GITHUB_PASSWORD=somepassword

RUN apk --no-cache add make git gcc libtool musl-dev

RUN mkdir -p /app
WORKDIR /app

# configure for private repositories
ENV GOPRIVATE=github.com/lavoqualis/*,github.com/kapigo/*
RUN echo "machine github.com login $GITHUB_USER password $GITHUB_PASSWORD" > $HOME/.netrc && \
    chmod 600 $HOME/.netrc
 
ARG APPNAME=promnats
ARG VERSION=0.0.0-dev   
# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

COPY . /app
RUN go build -ldflags "-s -w -X 'main.appVersion=${VERSION}'" -o /app/out/${APPNAME} ./cmd/promnats/


# running container
FROM alpine:3.18
LABEL org.opencontainers.image.base.name="alpine:3.18"
LABEL org.opencontainers.image.source https://github.com/kmpm/promnats.go

ARG USER=default

# install sudo as root
RUN apk add ca-certificates su-exec tzdata

# RUN adduser -D $USER
# RUN adduser -D $USER \
#         && echo "$USER ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/$USER \
#         && chmod 0440 /etc/sudoers.d/$USER

RUN mkdir -p /app/data
WORKDIR /app

ARG APPNAME=promnats
ARG VERSION=0.0.0-dev

COPY --from=builder /app/out/${APPNAME} /usr/local/bin/${APPNAME}
COPY entrypoint.sh /usr/local/bin/

RUN chmod +x /usr/local/bin/* 
EXPOSE 8083

ENTRYPOINT ["entrypoint.sh"]

CMD ["promnats"]