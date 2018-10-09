# Architecture

runmachine contains a set of services that provide a control plane for managing
hardware or virtual machines.

## Dependent services

runmachine depends on the following infrastructure services being installed:

* an `etcd` data store
* a relational database
* Gearman for simple job queueing

## runmachine services

### `runm-metadata`

The `runm-metadata` service has a number of purposes:

1) to translate a string name into a UUID external identifier and vice versa
2) to store and retrieve tag and key/value information for objects via a UUID
   external identifier
3) to allow administrators to define simple schema constraints on the values
   stored in items with particular keys
4) to allow administrators to define which classes of user and project may read
   or write items with particular keys

### `runm-resource`

gRPC service endpoint that is responsible for storing data about the following
objects in the system:

* resource class
* capability
* distance type
* distance
* provider type
* provider
* provider inventory
* provider group
* provider group distances
* consumer type
* consumer
* claim
* allocation
* usage

### `runm-account`

gRPC service endpoint that is responsible for storing data about the following
objects in the system:

* project
* user
* role
* quota limit

### `runm-control`

gRPC service endpoint that validates requests to perform some action, such as
provisioning a machine, and pushing a task onto a work queue to be grabbed by a
`runm-executor` worker.

### `runm-executor`

While not a service endpoint, a `runm-executor` is a process that reads a task
from a work queue and executes the steps involve in that task.
