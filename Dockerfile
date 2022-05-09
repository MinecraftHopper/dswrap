FROM rust:alpine AS builder

WORKDIR /build
COPY . .

RUN apk add musl-dev openssl-dev

ENV RUSTFLAGS="--emit=asm"
RUN cargo build --release

FROM alpine

COPY --from=builder /build/target/release/mcpaste /bin/mcpaste

CMD ["/bin/mcpaste"]
