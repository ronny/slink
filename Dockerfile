# Based on https://www.sethvargo.com/writing-github-actions-in-go/
#############################################################################
FROM golang:1.19 AS builder

RUN apt-get update && apt-get -y install upx

ENV CGO_ENABLED=0

WORKDIR /src

# These layers shouldn't change if there are no dependency changes
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN make binaries && \
    strip bin/* && \
    upx -q -9 bin/*

RUN echo "nobody:x:65534:65534:Nobody:/:" > /etc_passwd

#############################################################################
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc_passwd /etc/passwd
COPY --from=builder --chown=65534:0 /src/bin/ /bin/

USER nobody

ENV TZ=UTC

WORKDIR /

CMD ["/bin/slink-public-server"]
