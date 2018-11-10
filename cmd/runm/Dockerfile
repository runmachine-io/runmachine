# We use a multi-stage build, so we require Docker >=17.05 to build these
# images
FROM runm/base as builder
COPY . /go/src/github.com/runmachine-io/runmachine
WORKDIR /go/src/github.com/runmachine-io/runmachine/cmd/runm
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o /bin/runm .

# Take the built binary from the builder image and place it into a new
# from-scratch image, reducing the resulting image size substantially
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /bin/runm /bin/runm
WORKDIR /bin
CMD ["./runm"]
