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

To support highly-distributed hardware environments with many physical sites,
`runmachine` is designed from the beginning to be a shard-aware system. In
other words, the data housed in a particular installation of `runmachine` is
*always* contained in a unique partition, and data is stored in backend storage
systems along with that partitioning key.

Having this partitioning key stored in backend storage means that two
installations of `runmachine` can have identically-named objects (such as
providers named `site1.row1.rack1.node1`) and both of these `runmachine`
systems can share data with the other and handle each other's data without
worrying about violating uniqueness constraints within their own installation.

This enables many use cases, including federation (the ability for users of two
unique `runmachine` installations to operate on the other), data export/import
scenarios, and common "reseller" setups.

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

The natural counterpart to inventory management functionality is the ability
for users to consume that inventory in a controlled manner. Therefore, it is
essential that `runmachine` provide robust resource reservation and scheduling
along with transactionally safe resource claiming across the known set of
inventory.

In addition to an accurate view of the system's capacity, a good scheduling and
reservation system **must** have deep knowledge of the following in order to
make accurate and reliable placement decisions:

* capabilities/features of the providers of inventory
* how providers are grouped
* the relative distance between providers (important to satisfy affinity and
  anti-affinity constraints)

Furthermore, a reliable placement engine **must** be the system that handles
the atomic consumption of resources against the known system inventory. It is
*not* possible to build a reliable placement engine unless the resource-claim
process is fully owned and guaranteed by the placement engine itself. Too many
race conditions occur and too much system flakiness is observed when outside
agents are allowed to control all or part of the resource-claim process.

#### Structured object metadata and tagging

Most software needs to provide users (and the system itself) with the ability
to associate information with objects the system knows about. This information
is sometimes called "metadata" (data **about** data).

However, when there are no rules to how metadata can be used to catalog objects
in the system, maintainability and interoperability of the software is
decreased.

`runmachine` tries to balance the need for robust, full-featured cataloguing of
objects with these concerns about long-term code maintenance and
interoperability.

#### Start and stop machine resources

TODO

#### What `runmachine` IS NOT

While it is important to define the scope of a project, it's also helpful to
list some things that `runmachine` is **NOT** trying to be.

Here are things that `runmachine` has no interest in being:

* A container orchestration system

There are plenty of excellent container orchestration systems around. If what
you need is a system to describe how different application processes,
microservices daemons or application containers relate to each other, use one
of the excellent existing container orchestration systems (Kubernetes, Docker
Compose/Swarm and Mesos are all good fits).

* A job scheduling system

`runmachine` doesn't aim to provide some general job/batch scheduling system.
If you need this kind of functionality, look at Kubernetes (it has a Job object
that suits this use case) or Mesos.

* A hypervisor or virtualization abstraction system

We also have no interest in writing hypervisor software or in providing an
abstraction layer over multiple hypervisors. If you need a super-flexible,
vendor-neutral hypervisor abstraction layer, look at things like OpenStack
Nova or the libvirt/kubevirt projects.

* A traditional "enterprise" managed virtualization system

`runmachine` has no intention of filling use cases that traditional heavyweight
managed virtualization platforms like VMWare/vCenter provide. If you (think
you) need functionality around "live migration" or "live resize" or
"distributed resource scheduling", `runmachine` is definitely not something you
should look at. Instead, fork your money over to VMWare or try to use the
functionality provided by OpenStack Nova's vCenter virtualization driver.

## Architecture

`runmachine`'s architecture is detailed in a separate
[document](docs/architecture.md), however, the basic architecture of
`runmachine` is a small set of services that handle a distinct piece of
functionality. These services work together to provide a `runmachine` user the
ability to manage compute resources.

* `runm-metadata` is a service providing functionality for associating both
  structured and unstructured data with one or more objects.
* `runm-resource` is a service providing inventory management,
  placement/scheduling and usage information.
* `runm-control` is a service that validates a request to take some action and
  then forwards that action along to one or more `runm-exec` workers that
  complete the work
