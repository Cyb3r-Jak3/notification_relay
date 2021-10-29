FROM golang:1.17-alpine as build

WORKDIR /go/src/app
COPY . /go/src/app

RUN go get -d -v ./
ENV CGO_ENABLED=0

RUN go build cmd -o /go/bin/app

FROM gcr.io/distroless/static:latest
COPY --from=build /go/bin/app /.
ENTRYPOINT ["/app"]
CMD [ "$@" ]