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

## Credential

Credentials are what are supplied to an identity provider to identify the user
and determine the scope of authorized actions that user has in the target
system.

Credentials always include a *partition* and a *project* when communicating
with an identity provider. The combination of partition and project, along with
the user, allow the identity provider to determine the scope of authorized
actions.

## Partition

A **partition** is `runmachine`'s way of carving up the world into smaller
manageable divisions. It is comprised of a globally unique identifier (UUID)
and a human-readable name. A partition may describe a geographic location or an
entirely abstract division.

When interacting with a `runmachine` deployment, users *MUST* specify the
partition that they are interacting with. This does *NOT* mean that
`runmachine` can only execute operations against a single partition at a time.
Users are authorized to access or execute actions against a specific partition.

## Project

A **project**, provided by a user when performing actions against a
`runmachine` deployment, is a *globally-unique identifier* that indicates the
role or group that the user is acting as.

## Object

An object is something that has all of the following characteristics:

* It has external human-readable name that is unique within either the
  [scope](#Object Type Scope) of a partition or the combination of the
  partition and project
* It has a globally-unique identifier (UUID)
* It can have [properties](#Property) associated with it
* It can have [tags](#Tag) associated with it

## Object Type

In `runmachine` systems, objects all have a specific **object type**. An object
type is a well-known code, such as `runm.machine` along with a description of
what the type of object is used for.

# Object Type Scope

An object type is either *partition-scoped* or *project-scoped*.
Partition-scoped object's have human-readable names that are guaranteed to be
unique within a partition. Project-scoped objects have human-readable names
that are unique within a partition and project combination.

## Property

A *property* is simply a key/value pair.

An [object](#Object) may have zero or more properties associated with it.

Properties may have a [property definition](#Property Definition) that
constrains the values for the property.

## Tag

A *tag* is a simple string. [Objects](#Object) may have zero or more tags
associated with them. There are no constraints placed on a tag's string value.
Any user with read access to an object's partition (or project if the object is
[project-scoped](#Object Type Scope)) may view an object's tags. Any user with
write access to the object may view the object's tags.

## Property Definition

When creating or updating objects in the system, a user may associate one or
more [properties](#Properties) with that object. Administrators may set up
constraints on the type and format of the values of these properties by setting
a *property definition* for a specific partition, object type and property key
combination.

In setting a property's definition, the administrator can control the set of
values that may be used for a particular property as well as define which
groups of users may read or write the property.

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
