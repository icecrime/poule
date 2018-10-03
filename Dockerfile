FROM golang:1.7-alpine

RUN apk update && apk add make git
RUN go get github.com/constabulary/gb/...

ADD . /src
RUN cd /src && make

ENTRYPOINT [ "/src/bin/poule" ]
