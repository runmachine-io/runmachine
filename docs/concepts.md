# Concepts in `runmachine`

This documents various objects exposed by `runmachine` and critical concepts
involved in a `runmachine` deployment.

## Access control

**Access control** is the process of both *authenticating* a user as well as
*authorizing* that user to perform some action against the system.

`runmachine` does not implement its own authentication functionality. Instead,
it relies on an external *identity provider* to perform authentication and tell
`runmachine` what types of actions a *user* is permitted to take against a
`runmachine` system.

## Identity provider

The *identity provider* determines whether a supplied set of *credentials* is
allowed to interact with a `runmachine` deployment.

When performing actions against a `runmachine`, the user specifies a set of
*credentials* that are used to communicate with the identity provider.

## Credentials

Credentials are what are supplied to an identity provider to identify the user
and determine the scope of authorized actions that user has in the target
system.

Credentials always include a *partition* and a *project* when communicating
with an identity provider. The combination of partition and project, along with
the user, allow the identity provider to determine the scope of authorized
actions.

## Partitions

A **partition** is `runmachine`'s way of carving up the world into smaller
manageable divisions. It is comprised of a globally unique identifier (UUID)
and a human-readable name. A partition may describe a geographic location or an
entirely abstract division.

When interacting with a `runmachine` deployment, users *MUST* specify the
partition that they are interacting with. This does *NOT* mean that
`runmachine` can only execute operations against a single partition at a time.
Users are authorized to access or execute actions against a specific partition.

## Projects

A **project**, provided by a user when performing actions against a
`runmachine` deployment, is a *globally-unique identifier* that indicates the
role or group that the user is acting as.

## Objects

An object is something that has all of the following characteristics:

* It has external human-readable name that is unique within either the scope of
  a partition or the combination of the partition and project
* It has a globally-unique identifier (UUID)
* It can have *properties* associated with it
* It can have *tags* associated with it

## Object Types

In `runmachine` systems, objects all have a specific **object type**. An object
type is a well-known code, such as `runm.machine` along with a description of
what the type of object is used for.

## Properties

TODO

## Tags

TODO

## Property Schemas

TODO

## Machines

TODO

## Images

TODO

## Providers

TODO

## Provider Groups

TODO

## Capabilities

TODO

## Resource Types

TODO

## Inventories

TODO

## Claims

TODO

## Allocations

TODO
