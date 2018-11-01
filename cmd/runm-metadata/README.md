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

## Functionality provided by `runm-metadata`

TODO

### Name to UUID lookups

TODO

### Property schemas

TODO

### Object tagging

TODO
