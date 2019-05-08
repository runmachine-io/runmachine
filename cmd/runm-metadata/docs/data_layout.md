# Internal data layout for `runm-metadata`

The `runm-metadata` service uses an [etcd3](https://coreos.com/etcd/) key-value
store (KVS) for its backend data storage needs.

This document outlines how the keys are organized in `etcd` and how "indexes"
are built into the key layout so as to efficiently locate groups of keys by
prefix.

Before we describe the layout, it's good to go over some terminology.

## Key namespaces

When we use the term **key namespace**, we simply refer to a string key in
`etcd` that ends in a "/". These keys will have no value but rather will have a
set of subkeys "under" it. If you think of the `etcd` data store as a
filesystem, you can equate a key namespace as a directory in the filesystem.

When showing a key namespace, we use a directory-like structure, like so:

```
a/
  b/
    c/
    d/
      e
```

Above, `a/`, `a/b/`, `a/b/c/`, `a/b/d/` are all key namespaces, while `a/b/d/e`
represents a key.

## Valued keys

If you see a `{key} -> {value}` in the directory layout, that means that there
is a non-nil value for the key.

For instance, in the following graphic, the key `e` has a value stored of `x`:

```
a/
  b/
    c/
    d/
      e -> x
```

We refer to these types of keys that have a non-nil as **valued keys**. Valued
keys are often used by `runm-metadata` as a way to build **indexes** into the
underlying data, allowing fast lookup of information by non-UUID values.

## Understanding the key layout

The root of the tree of keys in `etcd` used by the `runm-metadata` service
always starts at the value designated by the
`RUNM_METADATA_STORAGE_ETCD_KEY_PREFIX` environs variable (or equivalent
`--storage-etcd-key-prefix` command line option) and is followed by the
constant `runm/metadata`.

So, for example, assuming that `RUNM_METADATA_STORAGE_ETCD_KEY_PREFIX` is its
default value of `/`. The top-level key namespace in `etcd` for all
`runm-metadata` data would be the string `runm/metadata/`:

```
runm/metadata/
```

This value will be referred to as the `$ROOT` key namespace (or just `$ROOT`
for short) in this document.

### The `$ROOT` key namespace

The keys directly under `$ROOT` describe the known **object types** and
**partitions** in the system. It's easier to explain the structure by looking
at a sample key namespace layout.

```
$ROOT
  definitions/
    by-type/
      runm.provider/
        default -> d296e062ef55443a8dd40369e2a3048d
        by-type/
          runm.compute -> c5e44b69fcd142dda035041df3967f11
          runm.storage.block -> 291defcaae2e4656ae47aee877b9a9ef
    by-uuid/
      291defcaae2e4656ae47aee877b9a9ef -> serialized ObjectDefinition message
      62026a2934c54df395ba44b0b398c808 -> serialized ObjectDefinition message
      c5e44b69fcd142dda035041df3967f11 -> serialized ObjectDefinition message
      d133ac2327ae49e8a4d72a1b57e1ed0c -> serialized ObjectDefinition message
      d296e062ef55443a8dd40369e2a3048d -> serialized ObjectDefinition message
      f3823e51dbfd420f807f3a1daac674f0 -> serialized ObjectDefinition message
  partitions/
    d3873f99a21f45f5bce156c1f8b84b03/
    d79706e01fbd4e48aae89209061cdb71/
  types/
    object/
      runm.image -> serialized ObjectType message
      runm.machine -> serialized ObjectType message
      runm.partition -> serialized ObjectType message
      runm.provider -> serialized ObjectType message
      runm.provider_group -> serialized ObjectType message
    runm.provider/
      runm.compute_node -> serialized ProviderType message
  objects/
    by-type/
      runm.partition/
        by-name/
          us-east.example.com -> d3873f99a21f45f5bce156c1f8b84b03
          us-west.example.com -> d79706e01fbd4e48aae89209061cdb71
    by-uuid/
      d3873f99a21f45f5bce156c1f8b84b03 -> serialized Object message
      d79706e01fbd4e48aae89209061cdb71 -> serialized Object message
      54b8d8d7e24c43799bbf70c16e921e52 -> serialized Object message
      60b53edd16764f6abc081ddb0a73e69c -> serialized Object message
      3bf3e700f11b4a7cb99244c554b3a856 -> serialized Object message
```

Above, you can see that `$ROOT` has a four top-level key namespaces:`types/`,
`objects/`, `partitions/` and `definitions/`. These top-level key namespaces
have sub key namespaces under them that either contain serialized protobuffer
messages or contain a UUID value that points to a `by-uuid` index containing a
protobuffer message.

The `$ROOT/types/` key namespace describes different types in the runmachine
system.

The `$ROOT/definitions/` key namespace contains object definitions (schemas)
that describe different types in the system.

The `$ROOT/objects/` key namespace contains actual objects known to the system.

Finally, the `$ROOT/partitions/` key namespace contains a number of indexes
that allows the metadata service to quickly determine if a queried-for object
is in a partition partition.

The partition key namespace for the partition with UUID
`d79706e01fbd4e48aae89209061cdb71` will always be
`$ROOT/partitions/d79706e01fbd4e48aae89209061cdb71/`.

We will refer to an **individual partition key namespace** as `$PARTITION` from
here on.

### The `$PARTITION` key namespace

Under `$PARTITION`, we store information about the definitions (schemas) that
have been set up for a partition as well as a number of name indexes for
objects contained in the partition.

```
$PARTITION (e.g. $ROOT/partitions/d79706e01fbd4e48aae89209061cdb71/)
  definitions/
  objects/
```

We will refer to the `$PARTITION/objects/` key namespace as `$OBJECT` from here
on. Similarly, we will refer to `$PARTITION/definitions/` as just
`$DEFINITIONS`.

### The `$DEFINITIONS` key namespace

`$DEFINITIONS` key namespace contains a sub key namespace called `by-type` that
contains indexes into object definitions by type.

```
$DEFINITIONS (e.g. $ROOT/partitions/d79706e01fbd4e48aae89209061cdb71/definitions/)
  by-type/
    runm.provider/
      default -> 62026a2934c54df395ba44b0b398c808
      by-type/
        runm.compute -> f3823e51dbfd420f807f3a1daac674f0
        runm.storage.block -> id133ac2327ae49e8a4d72a1b57e1ed0c
```

The "default" object definition provides the default schema and permissions for
objects of that type when an administrator **has overridden** a definition for
that object type in the partition. Further key namespaces may be defined if an
object type has a sub-type (such as a provider type in the above example) and
an administrator has defined an object definition for that sub-type.

### The `$OBJECTS` key namespace

The `$OBJECTS` key namespace contains a sub key namespace called `by-type`
that contains indexes into objects by type.

```
$OBJECTS (e.g. $ROOT/partitions/d79706e01fbd4e48aae89209061cdb71/objects/)
  by-type/
    runm.image/
      by-project/
        eff883565999408dbec3eb5070d5ecf5/
          by-name/
            rhel7.5.2 -> 54b8d8d7e24c43799bbf70c16e921e52
            debian-sid -> 60b53edd16764f6abc081ddb0a73e69c
    runm.provider_group/
      by-name/
        us-east1-row1-rack2 -> 3bf3e700f11b4a7cb99244c554b3a856
```

As you see above, the `$OBJECTS/by-type/` key namespace contains additional key
namespaces, arranged in an a series of indexes so that `runm-metadata` can look
up UUIDs of various objects of that type that have a particular name and
optionally belong to a specific project.

The example key layout above shows a partition that has two image objects named
`rhel7.5.2` and `debian-sid` in a project with the UUID
`eff883565999408dbec3eb5070d5ecf5`. There is also a `runm.provider_group` object named
`us-east1-row1-rack2` with the UUID of `3bf3e700f11b4a7cb99244c554b3a856`.
Provider groups are objects with an object type scope of `PARTITION` which
means that these objects are not specific to a project, and therefore the
`$OBJECTS/by-type/runm.provider_group/by-name` is the only index key namespace
for these types of objects (there is no `by-project/` sub key namespace under
the object type).
