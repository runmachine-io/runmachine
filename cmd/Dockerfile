# Dockerfile for a base builder image for all runmachine components
#
# We use a multi-stage build, so we require Docker >=17.05 to build these
# images.
FROM golang:alpine

# First, add our base packages to the image
RUN apk --no-cache add ca-certificates

# Next, copy all the vendored packages that dep has ensured into the GOPATH
COPY vendor/ /go/src/
