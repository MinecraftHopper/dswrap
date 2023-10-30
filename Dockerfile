FROM golang:alpine AS builder

WORKDIR /build
COPY . .

RUN go build -o dswrap github.com/minecrafthopper/dswrap

FROM alpine

WORKDIR /dswrap

COPY --from=builder /build/dswrap /dswrap/dswrap

ENV DISCORD_TOKEN="" \
    DISCORD_TOKEN_FILE=""

EXPOSE 8080

CMD ["/dswrap/dswrap"]
