FROM golang:1.14 AS builder

WORKDIR /go/src/ddns
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

FROM golang:1.14

WORKDIR /go/bin
COPY --from=builder /go/bin/server .

CMD ["server"]