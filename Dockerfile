FROM golang:latest as build

RUN mkdir -p /go/src/
WORKDIR /go/src/

COPY . /go/src/

RUN CGO_ENABLED=0 go build -a -installsuffix cgo --ldflags "-s -w" -o /usr/bin/server 

FROM alpine:3.7

COPY --from=build /usr/bin/server /root/

EXPOSE 8080
EXPOSE 9090
WORKDIR /root/

CMD ["./server"]
