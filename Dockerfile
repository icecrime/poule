FROM golang:1.7-alpine

RUN apk update && apk add make && apk add git
RUN go get github.com/constabulary/gb/...

ADD . /src
RUN cd /src && apk add make && make

ENTRYPOINT [ "/src/bin/poule" ]
