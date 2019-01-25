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
    by-uuid/
      62026a2934c54df395ba44b0b398c808 -> serialized ObjectDefinition message
      d133ac2327ae49e8a4d72a1b57e1ed0c -> serialized ObjectDefinition message
      d296e062ef55443a8dd40369e2a3048d -> serialized ObjectDefinition message
      f3823e51dbfd420f807f3a1daac674f0 -> serialized ObjectDefinition message
  partitions/
    by-name/
      us-east.example.com -> d3873f99a21f45f5bce156c1f8b84b03
      us-west.example.com -> d79706e01fbd4e48aae89209061cdb71
    by-uuid/
      d3873f99a21f45f5bce156c1f8b84b03 -> serialized Partition message
      d79706e01fbd4e48aae89209061cdb71 -> serialized Partition message
    d3873f99a21f45f5bce156c1f8b84b03/
    d79706e01fbd4e48aae89209061cdb71/
  types/
    object/
      runm.image -> serialized ObjectType message
      runm.machine -> serialized ObjectType message
      runm.provider -> serialized ObjectType message
      runm.provider_group -> serialized ObjectType message
    runm.provider/
      runm.compute_node -> serialized ProviderType message
  objects/by-uuid/
    54b8d8d7e24c43799bbf70c16e921e52 -> serialized Object message
    60b53edd16764f6abc081ddb0a73e69c -> serialized Object message
    3bf3e700f11b4a7cb99244c554b3a856 -> serialized Object message
```

Above, you can see that `$ROOT` has four key namespaces, one called
`types/`, one called `objects/by-uuid/` one called `partitions` and
another called `definitions/by-type/`.

The `$ROOT/types/` key namespace has a set of key namespaces, each of which has
a set of [valued keys](#Valued keys) describing the different types in the
runmachine system.

The `$ROOT/definitions/by-type/` key namespace has a set of key namespaces, one
for each object type. These are the default object definitions that are used
when an administrator hasn't set up object definitions specific to a partition
for that object type. The `$ROOT/definitions/by-uuid` key namespace has a set
of valued keys where the keys are UUIDs and the values are serialized
ObjectDefinition protobuffer messages.

The `$ROOT/objects/by-uuid/` key namespace has a set of valued keys describing
the objects known to the system.

The valued keys in the `$ROOT/objects/by-uuid/` key namespace have the UUID of
the object as the key and a serialized Google Protobuffer message of the
[Object](../../../proto/defs/object.proto) itself as the value.

**NOTE**: Having the serialized Object message as the value of the
`$ROOT/objects/by-uuid/` key namespace's valued keys allows the `runm-metadata`
service to answer queries like "get me the tags on this object" with an
efficient single key fetch operation.

The `$ROOT/partitions/` key namespace has two key namespaces below it that
implement indexes into partitions, called `by-name` and `by-uuid`. In addition
to those index key namespaces, each partition's objects are contained in a key
namespace `$ROOT/partitions/{uuid}/`.

The `$ROOT/partitions/by-name/` key namespace contains valued keys, with the
key being the human-readable name of the partition and the value being the UUID
of that partition.

Each UUID value listed in `$ROOT/partitions/by-name/` will be a key namespace
under `$ROOT/partitions/{uuid}/` that contains *all* objects known to that
partition. We call these key namespaces "partition key namespaces".

**NOTE**: Typically, clients interacting with `runm-metadata` automatically
inject a partition identifier into the session when communicating with a
`runmachine` component, so this partition identifier isn't usually something
that one specifies manually on the command line. Instead, a configuration file
or environment variable contains the human-readable name of the partition that
the user is communicating with.

Therefore, the partition key namespace for the partition with UUID
`d79706e01fbd4e48aae89209061cdb71` will always be
`$ROOT/partitions/d79706e01fbd4e48aae89209061cdb71/`.

We will refer to an **individual partition key namespace** as `$PARTITION` from
here on.

**NOTE**: It is important to point out that the following keys are *different*:

* `$ROOT/partitions/by-uuid/d79706e01fbd4e48aae89209061cdb71`
* `$ROOT/partitions/d79706e01fbd4e48aae89209061cdb71/`

The former is a valued key that will have as its value a serialized `Partition`
protobuffer object. The latter is the partition key namespace for the partition
with UUID `d79706e01fbd4e48aae89209061cdb71`.

### The `$PARTITION` key namespace

Under `$PARTITION`, we store information about the definitions (schemas),
objects, and the object metadata (properties and tags) in the partition:

```
$PARTITION (e.g. $ROOT/partitions/d79706e01fbd4e48aae89209061cdb71/)
  definitions/
  objects/
  properties/
  tags/
```

We will refer to the `$PARTITION/objects/` key namespace as `$OBJECT` from here
on. Similarly, we will refer to `$PARTITION/definitions/` as just
`$DEFINITIONS`, `$PARTITION/properties/` as just `$PROPERTIES` and
`$PARTITION/tags,` as just `$TAGS`. Each of these key namespaces is described
in detail in the following sections.

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

### The `$PROPERTIES` key namespace

The `$PROPERTIES` key namespace stores information about the object properties
for all objects known to the partition.

Here is an example layout for a partition with a two `runm.image` objects and
one `runm.machine` object, with the image objects having an "architecture"
property associated with them and the machine object having an "appgroup"
property associated with it:

```
$PROPERTIES (e.g. $ROOT/partitions/d79706e01fbd4e48aae89209061cdb71/properties/)
  by-type/
    runm.image/
      architecture/
        x86_64/
          54b8d8d7e24c43799bbf70c16e921e52
        arm64/
          60b53edd16764f6abc081ddb0a73e69c
    runm.machine/
      appgroup/
        A/
          3bf3e700f11b4a7cb99244c554b3a856
```

`runm-metadata` can use the key namespaces defined in the `$PROPERTIES` key
namespace to look up objects that have a particular property (key, or key and
value). The lowest-level keys are UUIDs for the objects that match that
particular property.

### The `$TAGS` key namespace

Finally, the `$TAGS` namespace contains all the simple string tags for objects
in a partition. The structure of this key namespace looks like this:

```
$TAGS (e.g. $ROOT/partitions/d79706e01fbd4e48aae89209061cdb71/tags/)
  unicorn/
    54b8d8d7e24c43799bbf70c16e921e52
    60b53edd16764f6abc081ddb0a73e69c
  rainbow/
    3bf3e700f11b4a7cb99244c554b3a856
```

Above, we have three objects in the partition that have tags decorating them.
Two objects are decorated with a tag "unicorn" and one object is decorated with
a tag "rainbow". The lowest-level keys within each tag key namespace are the
object's UUID.
