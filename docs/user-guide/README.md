# `runmachine` User Guide

This document is a guide for users of a `runmachine` system. If you are looking
for a guide to install and configure `runmachine`, please see the [Operator
Guide](../ops-guide).

## Installing the `runm` client

The `runm` command-line program is used to perform actions against a
`runmachine` deployment. Therefore, the first thing you will want to do is
install the `runm` client.

### Installing via `apt` on Debian/Ubuntu

TODO

### Installing via `yum` on RHEL/CentOS

TODO

### Installing a Docker image with the `runm` client

TODO

## Getting started with the `runm` client

The first step in using the `runm` client is to set a few environment variables
(or alternately, supply command-line options) with information about where a
`runmachine` server can be found.

TODO

## Administering a partition

### Viewing the default provider definition

To view the default [object definition](../concepts.md#object-definition) for a
[provider](../concepts.md#provider), use the `runm provider definition get`
command. It has three convenient calling conventions.

`runm provider definition get --global` returns the global default object
definition for providers.

`runm provider definition get --partition $PARTITION` returns the object
definition that has been overridden for providers in the specified
`$PARTITION`, or an empty string if the object definition for providers has not
been overridden for that partition.

Finally, `runm provider definition get` with no CLI options returns the object
definition that has been overridden *for the partition in user's session* OR
the global default if no override has been set for that partition.

Running `runm provider definition get` with no CLI options is the best way to
see the object definition that would be used when creating a new provider and
not passing in any partition information.

### Modifying the definition of providers

Administrators may wish to modify the default [object
definition](../concepts.md#object-definition) for
[providers](../concepts.md#provider) in a particular partition.

A common reason for doing so would be to ensure a particular piece of location
information (e.g. the site that a piece of hardware was physically in) is
always set on a provider object.

`runm provider definition set` is used to modify a provider definition.

### Provider definition example

Let's suppose Alice is an administrator for a partition called "part0".  Alice
wants to ensure that every time a [provider](../concepts.md#provider) is
created, that a "location.site" property is set for the machine. Furthermore,
she wants to ensure that this property is only visible to administrators.

Alice creates a file called "provider-def.yaml" with the following content:

```yaml
property_definitions:
  location.site:
    required: true
    permissions:
      # Default to not allowing reads from anyone
      - permission:
      # Unless the user is an admin
      - role: admin
        permission: rw
    schema:
      type: string
```

Alice then uses the `runm provider definition set` command to override the
provider definition for partition "part0":

```
runm provider definition set --partition part0 -f provider-def.yaml
```

After executing the `runm provider definition set` command, users attempting to
create a provider in partition "part0" would be required to set the property
with key "location.site" to a string value. Furthermore, non-administrator
users would not be able to view the "location.site" property for any providers
in partition "part0".
