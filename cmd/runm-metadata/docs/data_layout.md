# Internal data layout for `runm-metadata`

The `runm-metadata` service uses an [etcd3](https://coreos.com/etcd/) key-value
store (KVS) for its backend data storage needs.

This document outlines how the keys are organized in `etcd` and how "indexes"
are built into the key layout so as to efficiently locate groups of keys by
prefix.

## Key layout

The root of the tree of keys in `etcd` used by the `runm-metadata` service
always starts at the value designated by the
`RUNM_METADATA_STORAGE_ETCD_KEY_PREFIX` environs variable (or equivalent
`--storage-etcd-key-prefix` command line option). This value will be referred
to as the `$ROOT` key namespace (or just `$ROOT` for short) in this document.

All keys directly Under `$ROOT` are partition UUIDs. Because of this, all
objects that `runm-metadata` owns must always be contained within a single
partition, and objects are always identified using a partition identifier.

```
$ROOT
  /d3873f99a21f45f5bce156c1f8b84b03
  /d79706e01fbd4e48aae89209061cdb71
```

Above, the `etcd` keys under `$ROOT` all represent partitions. So,
`d3873f99a21f45f5bce156c1f8b84b03` and `d79706e01fbd4e48aae89209061cdb71` are
the two partitions that this deployment of `runmachine` knows about.

**NOTE**: Typically, clients interacting with `runm-metadata` automatically
inject a partition identifier into the session when communicating with a
`runmachine` component, so this partition identifier isn't usually something
that one specifies manually on the command line.

We will refer to an individual partition key namespace as `$PARTITION` from
here on.

Under `$PARTITION`, we store information about the objects and property schemas
in the partition:

```
$PARTITION (e.g. $ROOT/d79706e01fbd4e48aae89209061cdb71)
  /objects
  /property_schemas
```

We will refer to the `$PARTITION/objects` key namespace as `$OBJECT` from here
on. Similarly, we will refer to `$PARTITION/property_schemas` as just
`$PROPERTY_SCHEMAS`

## The `$OBJECTS` key namespace

Let's first take a look at what is contained in the `$OBJECTS` key
namespace. This key namespace contains three additional key namespaces, the
names of which indicate what the keys in that key namespace contain.

```
$OBJECTS
  /by-type
  /uuids
```

The `$OBJECTS/by-type` key namespace contains additional key namespaces,
arranged in an a series of indexes so that `runm-metadata` can look up UUIDs of
various objects of that type that belong to a project and have a particular
name.

Here's an example key layout for a partition that has two image objects named
`rhel7.5.2` and `debian-sid` in a project with the UUID
`eff883565999408dbec3eb5070d5ecf5`:

```
$OBJECTS
  /by-type
    /runm.image
      /by-project
        /eff883565999408dbec3eb5070d5ecf5
          /by-name
            /rhel7.5.2
              /54b8d8d7e24c43799bbf70c16e921e52
            /debian-sid
              /60b53edd16764f6abc081ddb0a73e69c
          /uuids
            /60b53edd16764f6abc081ddb0a73e69c
            /54b8d8d7e24c43799bbf70c16e921e52
      /uuids
        /54b8d8d7e24c43799bbf70c16e921e52
        /60b53edd16764f6abc081ddb0a73e69c
    ...
```

The `$OBJECTS/uuids` key namespace contains keys with UUID identifiers of all
objects in the partition, regardless of type.
