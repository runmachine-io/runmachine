# runm-metadata

Most software needs to provide users (and the system itself) with the ability
to associate information with objects the system knows about. This information
is sometimes called "metadata" (data **about** data).

However, when there are no rules to how metadata can be used to catalog objects
in the system, maintainability and interoperability of the software is
decreased.

`runmachine` tries to balance the need for robust, full-featured cataloguing of
objects with these concerns about long-term code maintenance and
interoperability. `runm-metadata` is the server component responsible for
providing this structured object metadata and tagging functionality.

## Background

Software systems are defined by the data they store, how that data is shared
between users and components, what rules are in place to transform that data
and decorate it with attributes that provide meaning to users of the system.

Software that doesn't allow either the end user or the system itself to
adequately describe and prescribe additional meaning to a piece of data is
software that is doomed to be stuck in a constant struggle to evolve the
underlying data model with more and more specific attributes about the object
the data represents.

These new underlying data model attributes inevitably end up being specific to
a few use cases but irrelevant to a larger number of cases. Each additional
attribute adds cognitive dissonance to end user, being one more concept the end
user needs to internalize and commit to memory about a particular object.

On the other hand, software that attempts to provide ultimate flexibility and
no rules about how metadata is applied to objects tends to devolve into an
unmaintainable mess.

Users blindly label objects with misspelled attributes.  Groups of users use
different terms to describe the same thing, making it difficult for external
systems to correlate and aggregate information. [Tribal knowledge](https://en.wikipedia.org/wiki/Tribal_knowledge)
flourishes in such situations, and that tribal knowledge ends up destroying the
maintainability of the system.

A balance is needed between the rigid process of underlying schema migrations
and the free-for-all world of ungoverned labeling of objects.

The `runm-metadata` service tries to achieve this balance.

## Concepts

The `runm-metadata` service provides a structured lookup service for object
metadata in the `runmachine` system. The term "metadata" unfortunately has been
a bit overloaded by software designers and doesn't really have a common
definition. So, let's start with some definitions that will be handy when
discussing the concepts of metadata.

First, a `runmachine` system acts on many ***objects***.  An object can be
considered a simple record of some *thing* that `runmachine` knows about.

All objects have a specific **type**. Examples of object types in `runmachine`
are `runm.machine` or `runm.project`.

All objects are identified by an **external identifier** that is **globally
unique**. We use UUID values for these external identifiers.

All objects have an **name** that is unique **within a particular scope**.
Typically this scope is the combination of a **partition** and a **project**.

In addition to their type, external identifier and name, all objects may have
**metadata** associated with them. There are two kinds of metadata that may be
associated with an object:

* A **tag** is a simple string
* A **property** is a key/value pair

Tags are simple. The have no restrictions on structure or format, and they may
be added or removed from an object by any user belonging to the project that
owns the object (or any user having `SUPER` privileges).

Properties, on the other hand, may have a **schema** associated with the
**key**. This schema can define the data type and format of the **value**
component, can restrict read and/or write access to property, and associates a
**version** with the property's schema so that the definition of that property
can evolve over time in a measured fashion.

### Name to UUID lookups

An extremely common operation that a user (or the system itself) must perform
is correlating a human-readable name with some identifier (such as a UUID).
Likewise, the reverse operation, of correlating a UUID with a human-readable
name, is extremely common.

`runm-metadata` provides this name to UUID and UUID to name translation
functionality as a service to other `runmachine` service components. This
enables all other `runmachine` components to exclusively store UUID identifiers
in their backend data stores, enabling more efficient storage and retrieval of
information for those components.

### Property schemas

Users with certain privileges may define **property schemas** for a specific
property **key**. This schema is a document that describes the data type
restrictions and format for the **value** component of the property.

For example, let's say that an administrator wanted objects of type
`runm.image` to require an "os" property be associated with it. Furthermore,
this "os" property would be restricted to one of the following string values:

* "linux/redhat"
* "linux/debian"
* "windows"

The administrator might create a property schema that looked like this:

```yaml
key: os
object_type: runm.image
schema:
  type: string
  enum:
    - linux/redhat
    - linux/debian
    - windows
  required: true
```

The `runmachine` system will now be able to use that property schema to guard
the possible values that are allowed in the "os" properties on `runm.image`
objects. In addition, property schema definitions which provide valuable
information for UIs and external systems that need to discover how and what
kind of data may be associated with different categories of objects that
`runmachine` knows about.

### Simple tagging

Object tags are super simple strings associated with any object. There are no
restrictions on who can read or write these strings nor are there restrictions
on what the strings look like. This makes simple tags convenient for quick and
dirty labeling of objects, but they can be prone to duplication, misspelling
and abuse by a group's tribal knowledge.
