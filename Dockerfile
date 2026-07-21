FROM golang:1.25-alpine AS build-env
WORKDIR /opt/workdir/

COPY go.mod go.sum ./
RUN go mod download

COPY ./ /opt/workdir/
RUN CGO_ENABLED=0 GOOS=linux go build -o aws_batch_exporter cmd/aws_batch_exporter.go

FROM alpine:3.23
RUN apk --no-cache add ca-certificates
RUN addgroup -S exporterg && adduser -S exporter -G exporterg
USER exporter
COPY --from=build-env /opt/workdir/aws_batch_exporter /opt/aws_batch_exporter
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/metrics || exit 1

ENTRYPOINT ["/opt/aws_batch_exporter"]
