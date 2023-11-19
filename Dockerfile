FROM golang:1.21-bookworm AS build

WORKDIR /usr/src/ps-http-sim

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go install -ldflags "-s -w" -trimpath -v github.com/mattrobenolt/ps-http-sim

FROM scratch AS local
COPY --from=build /go/bin/ps-http-sim /ps-http-sim

EXPOSE 8080
ENTRYPOINT ["/ps-http-sim", "-listen-addr=0.0.0.0"]

FROM scratch AS release
COPY ps-http-sim /

EXPOSE 8080
ENTRYPOINT ["/ps-http-sim", "-listen-addr=0.0.0.0"]
