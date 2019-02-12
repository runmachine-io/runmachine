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

* It is of a specific [type](#object-type)
* It has external human-readable name that is unique within either the
  [scope](#object-type-scope) of a partition or the combination of the
  partition and project
* It has a globally-unique identifier (UUID)
* It has zero or more [attributes](#object-attribute)
* It can have [properties](#object-property) associated with it
* It can have [tags](#tag) associated with it

### Object Type

In `runmachine` systems, objects all have a specific **object type**. An object
type is a well-known code, such as `runm.provider` or `runm.machine` along with
a description of what the type of object is used for.

#### Object Type Scope

An object type is either *partition-scoped* or *project-scoped*.
Partition-scoped objects have human-readable names that are guaranteed to be
unique within a partition. Project-scoped objects have human-readable names
that are unique within a partition and project combination.

### Object Attribute

An object *attribute* is a fixed field of an object that is always guaranteed
to exist for all objects of that object type.

Object attributes are referenced directly by their name, as opposed to
properties, which are referenced in an object's properties map.

Examples of an object attribute would be a `runm.provider`'s `provider_type`
attribute. All objects of type `runm.provider` are guaranteed to have an
attribute called `provider_type`.

### Object Property

An *object property* is simply a key/value pair.

All objects have an attribute called `properties` that is the *property map*
containing all the object's properties.

Properties may have a [property definition](#property-definition) that
constrains the values for that property, indicate whether the property is
required for objects of a certain type, and which users are able to read or
write the property.

## Object Definition

Each type of object in the `runmachine` system has an *object type definition*
that constrains the structure and access control of objects of that type.

An *object definition* contains information about:

* the primary [attributes](#object-attribute) of that type of object, including
  the data type of the attribute and whether or not the attribute is required
* the keys and data types of [properties](#object-property) that may be set on
  objects of that type
* the access permissions that indicate who may read or write certain attributes
  and properties on objects of that type

Each type of object has a default object definition. These default object
definitions may be overridden for a specific [partition](#partition).

## Tag

A *tag* is a simple string. [Objects](#object) may have zero or more tags
associated with them. There are no constraints placed on a tag's string value.
Any user with read access to an object's partition (or project if the object is
[project-scoped](#object-type-scope)) may view an object's tags. Any user with
write access to the object may view the object's tags.

## Machine

TODO

## Image

TODO

## Provider

A *provider* is an [object](#object) in a `runmachine` that describes something
that provides resources.

Providers have a set of [capabilities](#capability) and exposes one or more
[inventories](#inventory) of [resources](#resource-type) that may be used by a
[consumer](#consumer).

Providers have a well-known provider type.

### Provider type

A category of [provider](#provider).

Well-known provider types include:

* `runm.compute`: A provider of compute resources (CPU, memory, local disk,
  etc)
* `runm.storage.block`: A provider of raw block storage

## Provider Group

TODO

## Capability

TODO

## Resource Type

TODO

## Inventory

TODO

## Consumer

TODO

## Claim

TODO

## Allocation

TODO
