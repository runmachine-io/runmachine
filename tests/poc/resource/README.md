# runm-resource data model testing

This directory contains Python and SQL code that tests the database schema and
modeling for the `runm-resource` service. The `runm-resource` service is a
critical component in `runmachine` and is responsible for resource accounting,
scheduling/placement and resource reservation. Therefore, we want to ensure
that the underlying database modeling is sound and that the queries required by
the service are performant, utilize indexes appropriately, and work well at
varying scales (of deployment size).

We test the data model and schema in Python because it's quick and easy to
prototype things and run tests. We're not interested in comparing the raw speed
of Golang versus Python. Rather, we're simply interested in quickly loading up
a DB with varying scale scenarios, running claim and placement tests aghainst
that DB, and tearing it all down.

We use MySQL for the tests, though there's nothing preventing the use of
PostgreSQL or another DB system. Again, we're not comparing MySQL vs
PostgreSQL. We're only interested in how the data model stands up under varying
query scenarios and deployment scales.

## How to use this code

**PREREQUISITES**: Install MySQL on your test machine. Make a note of the root
database password.

First, create a `virtualenv` so that you don't need to worry about Python
package dependencies on your local testing machine:

```
cd $ROOT_DIR/tests/poc
virtualenv .venv
```

Then activate your virtualenv and export your local database :

```
source .venv/bin/activate
```

Install the Python package dependencies into your virtualenv:

```
pip install -r requirements.txt
```

Export the password for the `root` user of your database:

```
export RUNM_TEST_RESOURCE_DB_PASS=foo
```

Run the resource data model test:

```
python resource/run.py --reset
```

**NOTE**: The `--reset` argument reloads the resource database. You only need
to run with `--reset` when you want to load (or re-load) a certain deployment
configuration (the `--deployment-config` CLI option can be used to switch
deployment configuration profiles)

The tool will load the database up with the deployment configuration described
by a YAML file and selected with the `--deployment-config` CLI option and
execute a single claim request described by a YAML file and selected with the
`--claim-config` CLI option.

Example output:

```
(.venv) [jaypipes@uberbox poc]$ python resource/run.py --reset \
> --deployment-config 10k-shared-compute \
> --claim-config 1cpu-64M-10G
loading deployment config ... ok
resetting resource PoC database ... ok
creating object types ... ok
creating provider types ... ok
creating resource classes ... ok
creating consumer types ... ok
creating capabilities ... ok
creating distance types ... ok
creating distances ... ok
creating partitions ... ok
creating provider groups ... ok
caching provider group internal IDs ... ok
caching resource class and capability internal IDs ... ok
caching partition, distance type and distance internal IDs ... ok
creating providers ... ok
loading claim config ... ok
Found 50 providers with capacity for 1000000000 runm.block_storage
Found 50 providers with capacity for 67108864 runm.memory
Found 50 providers with capacity for 1 runm.cpu.shared
Claim(allocation=
    Allocation(consumer=Consumer(name=instance0,uuid=e8495d7172714de0a0106e6a4c4927f7),claim_time=1540490434,release_time=9223372036854775807,items=[
        AllocationItem(provider=Provider(uuid=00af7f6b00224f81acea148e3318fe34),resource_class=runm.block_storage,used=1000000000),
        AllocationItem(provider=Provider(uuid=00af7f6b00224f81acea148e3318fe34),resource_class=runm.memory,used=67108864),
        AllocationItem(provider=Provider(uuid=00af7f6b00224f81acea148e3318fe34),resource_class=runm.cpu.shared,used=1)]))

```
