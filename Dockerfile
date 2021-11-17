FROM golang:1.17 as build-env

WORKDIR /go/src/fluenthose
COPY go.mod go.sum ./
COPY main.go main.go
COPY pkg/ pkg/
COPY cmd/ cmd/
RUN go mod download
RUN go vet -v

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -ldflags="-w -s" -a -o /go/bin/fluenthose main.go

FROM gcr.io/distroless/base:nonroot-amd64
ENV DISTRO="debian"
ENV GOARCH="amd64"
COPY --from=build-env /go/bin/fluenthose /
ENTRYPOINT ["/fluenthose"]