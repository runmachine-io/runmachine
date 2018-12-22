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

## Administering a project

### Controlling object metadata with property definitions

All objects in a `runmachine` system may be decorated with various pieces of
information. Depending on the deployer's preferences and internal policies,
administrators may wish to constrain the type and format of this information.
Administrators may also wish to prevent certain parties from changing or even
viewing specific pieces of information about objects.

Administrators are able to constrain object information using [property
definitions](../concepts.md#property-definition). A property definition may be
created for any [object type](../concepts.md#object-type) and
[property](../concepts.md#property) combination.

For example, let's suppose Alice is an administrator for a project called
"Project X". Alice wants to ensure that every time a `runm`
[machine](../concepts.md#machine) is created, that an "owner_email" property is
set for the machine. Furthermore, she wants to ensure that the value of this
"owner_email" property conforms to an internationalized email address.

Alice would use the `runm property-definition set` command, passing in a YAML
document that describes the schema and/or access permissions that should
constrain the "owner_email" property of the `runm.machine` object type, like
so:

```
cat <<EOF | runm property-definition set
partition: part0
type: runm.machine
key: owner_email
required: false
schema:
  type: string
  format: idn-email
  minLength: 5
  maxLength: 255
EOF
```

**NOTE**: Alternately, Alice could have used the `-f <FILE>` CLI option, like so:

```
$ cat runm.machine-owner_email.yaml
partition: part0
type: runm.machine
key: owner_email
required: false
schema:
  type: string
  format: idn-email
  minLength: 5
  maxLength: 255
$ runm property-definition set -f runm.machine-owner_email.yaml
ok
```

After executing the `runm property-definition set` command, users attempting to
create machine and not specifying a property called "owner_email" with a valid
internationalized email address would receive a validation failure.
