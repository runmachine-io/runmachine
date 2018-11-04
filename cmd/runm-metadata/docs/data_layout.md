# Internal data layout for `runm-metadata`

The `runm-metadata` service uses an [etcd3](https://coreos.com/etcd/) key-value
store (KVS) for its backend data storage needs.

This document outlines how the keys are organized in `etcd` and how "indexes"
are built into the key layout so as to efficiently locate groups of keys by
prefix.

Before we describe the layout, it's good to go over some terminology.

## Key namespaces

When we use the term **key namespace**, we simply refer to a string key in
`etcd` that has no value but rather has a set of subkeys "under" it. If you
think of the `etcd` data store as a filesystem, you can equate a key namespace
as a directory in the filesystem.

So, a key of `/a/b` is a key namspace if there exists additional keys (or key
namespaces) *under* `/a/b`, such as `/a/b/c` or `/a/b/d/e`.

When showing a key namespace, we use a directory-like structure, like so:

```
/a
  /b
    /c
    /d
      /e
```

## Valued keys

If you see a `{key} -> {value}` in the directory layout, that means that there
is a non-nil value for the key.

For instance, in the following graphic, the key `e` has a value stored of `x`:

```
/a
  /b
    /c
    /d
      /e -> x
```

We refer to these types of keys that have a non-nil as **valued keys**. Valued
keys are often used by `runm-metadata` as a way to build **indexes** into the
underlying data, allowing fast lookup of information by non-UUID values.

## Understanding the key layout

The root of the tree of keys in `etcd` used by the `runm-metadata` service
always starts at the value designated by the
`RUNM_METADATA_STORAGE_ETCD_KEY_PREFIX` environs variable (or equivalent
`--storage-etcd-key-prefix` command line option) and is followed by the
constant `runm-metadata`.

So, for example, assuming that `RUNM_METADATA_STORAGE_ETCD_KEY_PREFIX` is its
default value of `/`. The top-level key namespace in `etcd` for all
`runm-metadata` data would be the string `/runm-metadata`:

```
/runm-metadata
```

This value will be referred to as the `$ROOT` key namespace (or just `$ROOT`
for short) in this document.

### The `$ROOT` key namespace

The keys directly Under `$ROOT` describe the known **partitions** in the
system. It's easier to explain the structure by looking at a sample key
namespace layout.

```
$ROOT
  /partitions
    /by-name
      /us-east.example.com -> d3873f99a21f45f5bce156c1f8b84b03
      /us-west.example.com -> d79706e01fbd4e48aae89209061cdb71
    /by-uuid
      /d3873f99a21f45f5bce156c1f8b84b03
      /d79706e01fbd4e48aae89209061cdb71
```

Above, you can see that `$ROOT` has a single key namespace called `partitions`.
This key has two key namespaces below it, called `by-name` and `by-uuid`.

The `$ROOT/partitions/by-name` key namespace contains [valued keys](#Valued
keys), with the key being the human-readable name of the partition and the
value being the UUID of that partition.

Each UUID value listed in `$ROOT/partitions/by-name` will be a key namespace
under `$ROOT/partitions/by-uuid` that contains *all* objects known to that
partition. We call these key namespaces "partition key namespaces".

**NOTE**: Typically, clients interacting with `runm-metadata` automatically
inject a partition identifier into the session when communicating with a
`runmachine` component, so this partition identifier isn't usually something
that one specifies manually on the command line. Instead, a configuration file
or environment variable contains the human-readable name of the partition that
the user is communicating with.

Therefore, the partition key namespace for the partition with UUID
`d79706e01fbd4e48aae89209061cdb71` will always be
`$ROOT/partitions/by-uuid/d79706e01fbd4e48aae89209061cdb71`.

We will refer to an **individual partition key namespace** as `$PARTITION` from
here on.

### The `$PARTITION` key namespace

Under `$PARTITION`, we store information about the objects, property schemas,
and the object metadata (properties and tags) in the partition:

```
$PARTITION (e.g. $ROOT/partitions/by-uuid/d79706e01fbd4e48aae89209061cdb71)
  /objects
  /property-schemas
  /properties
  /tags
```

We will refer to the `$PARTITION/objects` key namespace as `$OBJECT` from here
on. Similarly, we will refer to `$PARTITION/property-schemas` as just
`$PROPERTY_SCHEMAS`, `$PARTITION/properties` as just `$PROPERTIES` and
`$PARTITION/tags` as just `$TAGS`. Each of these key namespaces is described in
detail in the following sections.

### The `$OBJECTS` key namespace

Let's first take a look at what is contained in the `$OBJECTS` key
namespace. Similar to the `$ROOT/partitions` key namespace, the `$OBJECTS` key
namespace contains two key namespaces called `by-type` and `by-uuid`:

```
$OBJECTS (e.g. $ROOT/partitions/by-uuid/d79706e01fbd4e48aae89209061cdb71/objects)
  /by-type
    /runm.image
      /by-project
        /eff883565999408dbec3eb5070d5ecf5
          /by-name
            /rhel7.5.2 -> 54b8d8d7e24c43799bbf70c16e921e52
            /debian-sid -> 60b53edd16764f6abc081ddb0a73e69c
  /by-uuid
    /54b8d8d7e24c43799bbf70c16e921e52
    /60b53edd16764f6abc081ddb0a73e69c
    ...
```

As you see above, the `$OBJECTS/by-type` key namespace contains additional key
namespaces, arranged in an a series of indexes so that `runm-metadata` can look
up UUIDs of various objects of that type that belong to a project and have a
particular name.

The example key layout above shows a partition that has two image objects named
`rhel7.5.2` and `debian-sid` in a project with the UUID
`eff883565999408dbec3eb5070d5ecf5`.

### The `$PROPERTY_SCHEMAS` key namespace

TODO

### The `$PROPERTIES` key namespace

TODO

### The `$TAGS` key namespace

TODO
