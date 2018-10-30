# Object models and service gRPC APIs

This directory contains the [Google Protocol
Buffer](https://developers.google.com/protocol-buffers/) files that define the
objects in the `runmachine` system as well as the
[gRPC](https://grpc.io://grpc.io/) APIs for each service component of
`runmachine`.

## Objects

Object definition `.proto` files describe some concept within `runmachine`.

If you're wondering about a particular term or concept in `runmachine`, a good
way to learn about that concept is to examine the object definition file for
the concept. There are comments within each object definition file that outline
what the object entails and how it relates to other objects in the system.

For example, if you're curious what a "provider" is, check out the comments in
the [proto/defs/provider.proto](provider.proto) file.

## Service APIs

`.proto` files that begin with the prefix `service_` define the gRPC service
APIs for a service component of `runmachine`. For example, the
[proto/defs/service_resource.proto](service_resource.proto) file describes the
`runm-resource` service component's interfaces for tracking and claiming
resources within a `runmachine` system.

## Evolution of object models and gRPC service APIs

**NOTE**: Until we hit a 0.1 milestone, we are not being strict about
backwards-incompatible object or service API changes. Things are in very fluid
in these early stages of development. Object models and service APIs are being
stretched and changed as we prototype the services and, more importantly, how
clients will interact and use these objects and service APIs.
