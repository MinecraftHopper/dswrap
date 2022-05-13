FROM golang:alpine AS builder

WORKDIR /build
COPY ./main.go ./main.go

RUN go build -o dswrap ./main.go

FROM alpine

WORKDIR /dswrap

COPY --from=builder /build/dswrap /dswrap/dswrap
COPY ./404.html ./paste.html /dswrap/

EXPOSE 8080

CMD ["/dswrap/dswrap"]
