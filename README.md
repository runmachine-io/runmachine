# runmachine

`runmachine` contains a set of services that provide a control plane for
managing hardware or virtual machines.

## Installation

TODO

## Basic usage

TODO

## About

Software for managing large collections of heterogeneous compute resources too
often is so highly customized for a particular vendor or operator that the
software itself becomes its own worst enemy. Code bloat, feature creep, and
deployment chaos are commonplace.

`runmachine` is a project that is borne from the ideas of developers and
operators who are tired of infrastructure management software that is overly
complex, impossible to maintain and difficult to reason about.

### Guiding principles

* Multi-tenant isolation

It is important that `runmachine` components be designed from the very
beginning with multi-tenancy in mind. By multi-tenancy, we refer to the ability
of the software to isolate one users' view and consumption of resources
from another user.

Application programming interfaces (APIs) should show only information
applicable or owned by the calling user and project. Likewise, any API that
allows a user to consume or reserve resources in the system should be governed
by a quota system, allowing administrators of the system to divide resources
fairly among users.

* No extensibility for the sake of extensibility

APIs should have clear purpose and operate on a well-defined set of data.

`runmachine` is *not* intended to be a framework for creating other
applications. Likewise, different installations of the same version of
`runmachine` should **always** publish **identical** APIs.

No API extensions. No custom API resources.

* Partitioning

TODO

### Project scope

It's important to keep the scope of any project tight and well-defined.
`runmachine` is no different. We do *not* intend `runmachine` to be some be-all
end-all enterprise application designed to solve all IT infrastructure
problems.

Instead, `runmachine` expose APIs that provide the following functionality:

* Inventory discovery and management
* Resource claims, reservations and scheduling
* Structured object metadata and tagging
* Start and stop machine resources

#### Inventory discovery and management

Data is often siloed and/or duplicated into different organizational
applications, with each department building hacky interfaces to munge, collect,
merge and replicate that data. As this data makes its way around the various
internal systems, different teams tack their own tribal knowledge onto the
data, transforming it to suit their own needs.

Eventually, nobody really knows who owns what data, what is the source of truth
about certain types of information, and which of several copies or alterations
of a piece of data to consider official.

One of `runmachine`'s primary purposes is to provide a clean, robust and
scalable way to discover and manage hardware inventory. `runmachine` should be
able to be *the system of record* for tracking vast amounts of heterogenous
hardware in an organization.

#### Resource claims, reservations and scheduling

TODO

#### Structure object metadata and tagging

TODO

#### Start and stop machine resources

TODO

## Architecture

TODO
