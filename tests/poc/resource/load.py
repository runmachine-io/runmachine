# Loads up the runm-resource database with records that we then use in our PoC
# scenarios

import os
import subprocess

import sqlalchemy as sa

import resource_models

_RESOURCE_SCHEMA_FILE = os.path.join(
    os.path.abspath(os.path.dirname(__file__)), 'resource_schema.sql',
)
_OBJECT_TYPE_MAP = None
_PROVIDER_TYPE_MAP = None


def _insert_records(tbl, recs):
    sess = resource_models.get_session()
    for rec in recs:
        ins = tbl.insert().values(**rec)
        sess.execute(ins)
    sess.commit()


def reset_db(ctx):
    ctx.status("resetting resource PoC database")
    db_user = os.environ.get('RUNM_TEST_RESOURCE_DB_USER', 'root')
    db_pass = os.environ.get('RUNM_TEST_RESOURCE_DB_PASS', '')
    cmd = "mysql -u%s -p%s 2>/dev/null < %s" % (
        db_user, db_pass, _RESOURCE_SCHEMA_FILE,
    )
    subprocess.check_output(cmd, shell=True)
    ctx.status_ok()


def create_object_types(ctx):
    ctx.status("creating object types")
    tbl = resource_models.get_table('object_types')

    recs = [
        dict(
            code="runm.partition",
            description="A division of resources. A deployment unit for runm",
        ),
        dict(
            code="runm.provider",
            description="A provider of some resources, e.g. a compute node or "
                        "an SR-IOV NIC",
        ),
        dict(
            code="runm.provider_group",
            description="A group of providers",
        ),
        dict(
            code="runm.image",
            description="A bootable bunch of bits",
        ),
        dict(
            code="runm.machine",
            description="Created by a user, a machine consumes compute "
                        "resources from one of more providers",
        ),
    ]
    try:
        _insert_records(tbl, recs)
        ctx.status_ok()
    except Exception as err:
        ctx.status_fail(err)


def get_object_type_map():
    """Returns a dict, keyed by object type string code, of internal object
    type ID.
    """
    global _OBJECT_TYPE_MAP
    if _OBJECT_TYPE_MAP is not None:
        return _OBJECT_TYPE_MAP
    tbl = resource_models.get_table('object_types')
    sel = sa.select([tbl.c.id, tbl.c.code])
    sess = resource_models.get_session()
    _OBJECT_TYPE_MAP = {r[1]: r[0] for r in sess.execute(sel)}
    return _OBJECT_TYPE_MAP


def create_resource_classes(ctx):
    ctx.status("creating resource classes")
    tbl = resource_models.get_table('resource_classes')

    recs = [
        dict(
            code="runm.cpu.dedicated",
            description="A logical CPU processor associated with a "
                        "single dedicated host CPU processor"
        ),
        dict(
            code="runm.cpu.shared",
            description="A logical CPU processor that may be executed on "
                        "a host CPU processor along with other shared logical "
                        "CPUs"
        ),
        dict(code='runm.memory', description='Bytes of RAM'),
        dict(code='runm.block_storage', description='Bytes of block storage'),
        dict(code='runm.gpu.virtual', description='virtual GPU context'),
    ]
    try:
        _insert_records(tbl, recs)
        ctx.status_ok()
    except Exception as err:
        ctx.status_fail(err)


def create_provider_types(ctx):
    ctx.status("creating provider types")
    tbl = resource_models.get_table('provider_types')

    recs = [
        dict(
            code="runm.compute",
            description="A provider of compute resources like CPU, memory, "
                        "etc",
        ),
        dict(
            code="runm.storage",
            description="A provider of generic disk resources",
        ),
        dict(
            code="runm.nic",
            description="A provider of network interface resources",
        ),
    ]
    try:
        _insert_records(tbl, recs)
        ctx.status_ok()
    except Exception as err:
        ctx.status_fail(err)


def get_provider_type_map():
    """Returns a dict, keyed by provider type string code, of internal provider
    type ID.
    """
    global _PROVIDER_TYPE_MAP
    if _PROVIDER_TYPE_MAP is not None:
        return _PROVIDER_TYPE_MAP
    tbl = resource_models.get_table('provider_types')
    sel = sa.select([tbl.c.id, tbl.c.code])
    sess = resource_models.get_session()
    _PROVIDER_TYPE_MAP = {r[1]: r[0] for r in sess.execute(sel)}
    return _PROVIDER_TYPE_MAP


def create_consumer_types(ctx):
    ctx.status("creating consumer types")
    tbl = resource_models.get_table('consumer_types')

    recs = [
        dict(
            code="runm.machine",
            description="A virtual or baremetal machine",
        ),
        dict(
            code="runm.volume",
            description="A persistent volume",
        ),
    ]
    try:
        _insert_records(tbl, recs)
        ctx.status_ok()
    except Exception as err:
        ctx.status_fail(err)


def create_capabilities(ctx):
    ctx.status("creating capabilities")
    tbl = resource_models.get_table('capabilities')

    recs = [
        dict(
            code="hw.cpu.x86.avx",
            description="Intel x86 CPU instruction set extensions for AVX",
        ),
        dict(
            code="hw.cpu.x86.avx2",
            description="Intel x86 CPU instruction set extensions for AVX2",
        ),
        dict(
            code="hw.cpu.x86.vmx",
            description="Intel x86 CPU instruction set extensions for VMX",
        ),
        dict(
            code="storage.disk.hdd",
            description="Block storage is on traditional spinning rust",
        ),
        dict(
            code="storage.disk.ssd",
            description="Block storage is on a solid-state drive",
        ),
    ]
    try:
        _insert_records(tbl, recs)
        ctx.status_ok()
    except Exception as err:
        ctx.status_fail(err)


def create_distances(ctx):
    ctx.status("creating distance types")
    dt_tbl = resource_models.get_table('distance_types')
    d_tbl = resource_models.get_table('distances')

    recs = [
        dict(
            code="network",
            description="Relative network distances",
            generation=1,
        ),
        dict(
            code="storage",
            description="Relative storage distances",
            generation=1,
        ),
        dict(
            code="failure",
            description="Relative distance representing successively smaller "
                        "chance of failure affecting workloads running on "
                        "that provider",
            generation=1,
        ),
    ]
    try:
        _insert_records(dt_tbl, recs)
        ctx.status_ok()
    except Exception as err:
        ctx.status_fail(err)

    ctx.status("creating distances")

    sess = resource_models.get_session()
    sel = sa.select([dt_tbl.c.id]).where(dt_tbl.c.code == "network")
    net_dt_id = sess.execute(sel).fetchone()[0]

    sel = sa.select([dt_tbl.c.id]).where(dt_tbl.c.code == "storage")
    storage_dt_id = sess.execute(sel).fetchone()[0]

    sel = sa.select([dt_tbl.c.id]).where(dt_tbl.c.code == "failure")
    failure_dt_id = sess.execute(sel).fetchone()[0]

    recs = [
        dict(
            type_id=net_dt_id,
            code="local",
            position=0,
            description="Very little network latency - within rack L2 subnet",
        ),
        dict(
            type_id=net_dt_id,
            code="site",
            position=1,
            description="Slightly higher latency between leaf "
                        "switch-connected nodes in a DC",
        ),
        dict(
            type_id=net_dt_id,
            code="remote",
            position=2,
            description="WAN latency",
        ),
        dict(
            type_id=storage_dt_id,
            code="local",
            position=0,
            description="Block storage local to host running machine",
        ),
        dict(
            type_id=storage_dt_id,
            code="row",
            position=1,
            description="NAS server shared by row of compute",
        ),
        dict(
            type_id=storage_dt_id,
            code="remote",
            position=2,
            description="External cloud block storage with WAN latency",
        ),
        dict(
            type_id=failure_dt_id,
            code="local",
            position=0,
            description="Providers are within same rack failure domain",
        ),
        dict(
            type_id=failure_dt_id,
            code="rack",
            position=1,
            description="Providers are in different rack power/failure domain",
        ),
        dict(
            type_id=failure_dt_id,
            code="row",
            position=2,
            description="Providers are in different row power/failure domain",
        ),
        dict(
            type_id=failure_dt_id,
            code="site",
            position=3,
            description="Providers are in different sites",
        ),
    ]
    try:
        _insert_records(d_tbl, recs)
        ctx.status_ok()
    except Exception as err:
        ctx.status_fail(err)


def create_partitions(ctx):
    ctx.status("creating partitions")
    obj_tbl = resource_models.get_table('object_names')
    part_tbl = resource_models.get_table('partitions')
    object_type_id = get_object_type_map()['runm.partition']

    sess = resource_models.get_session()

    created = set()

    try:
        for p in ctx.deployment_config.providers.values():
            part_uuid = p.partition.uuid
            if part_uuid in created:
                continue
            # Create the object lookup record
            obj_rec = dict(
                object_type=object_type_id,
                uuid=part_uuid,
                name=p.partition.name,
            )
            ins = obj_tbl.insert().values(**obj_rec)
            sess.execute(ins)

            # Create the base provider group record
            part_rec = dict(
                uuid=part_uuid,
            )
            ins = part_tbl.insert().values(**part_rec)
            sess.execute(ins)
            created.add(part_uuid)

        sess.commit()
        ctx.status_ok()
    except Exception as err:
        sess.rollback()
        ctx.status_fail(err)


def create_provider_groups(ctx):
    ctx.status("creating provider groups")
    obj_tbl = resource_models.get_table('object_names')
    pg_tbl = resource_models.get_table('provider_groups')
    object_type_id = get_object_type_map()['runm.provider_group']

    sess = resource_models.get_session()

    try:
        for pg in ctx.deployment_config.provider_groups.values():
            # Create the object lookup record
            obj_rec = dict(
                object_type=object_type_id,
                uuid=pg.uuid,
                name=pg.name,
            )
            ins = obj_tbl.insert().values(**obj_rec)
            sess.execute(ins)

            # Create the base provider group record
            pg_rec = dict(
                uuid=pg.uuid,
            )
            ins = pg_tbl.insert().values(**pg_rec)
            sess.execute(ins)

        sess.commit()
        ctx.status_ok()
    except Exception as err:
        sess.rollback()
        ctx.status_fail(err)


def create_providers(ctx):
    obj_tbl = resource_models.get_table('object_names')
    rc_tbl = resource_models.get_table('resource_classes')
    cap_tbl = resource_models.get_table('capabilities')
    part_tbl = resource_models.get_table('partitions')
    pg_tbl = resource_models.get_table('provider_groups')
    pg_members_tbl = resource_models.get_table('provider_group_members')
    p_tbl = resource_models.get_table('providers')
    p_caps_tbl = resource_models.get_table('provider_capabilities')
    tree_tbl = resource_models.get_table('provider_trees')
    inv_tbl = resource_models.get_table('inventories')
    pd_tbl = resource_models.get_table('provider_distances')
    dt_tbl = resource_models.get_table('distance_types')
    d_tbl = resource_models.get_table('distances')

    # in-process cache of partition name -> internal ID
    part_ids = {}
    # in-process cache of provider group name -> internal ID
    pg_ids = {}
    # in-process cache of resource class code -> internal ID
    rc_ids = {}
    # in-process cache of capability code -> internal ID
    cap_ids = {}
    # Hashmap of distance type code to internal ID
    distance_type_ids = {}
    # Hashmap of (distance_type_code, distance_code) to internal ID
    distance_ids = {}

    sess = resource_models.get_session()
    ctx.status("caching provider group internal IDs")
    for pg in ctx.deployment_config.provider_groups.values():
        if pg.uuid in pg_ids:
            pg_id = pg_ids[pg.uuid]
        else:
            # Not yet in cache... look up the provider group by UUID
            sel = sa.select([pg_tbl.c.id]).where(pg_tbl.c.uuid == pg.uuid)
            res = sess.execute(sel).fetchone()
            pg_ids[pg.uuid] = res[0]
    ctx.status_ok()

    ctx.status("caching resource class and capability internal IDs")
    for prof in ctx.deployment_config.profiles.values():
        for rc_code in prof['inventory'].keys():
            if rc_code not in rc_ids:
                sel = sa.select([rc_tbl.c.id]).where(rc_tbl.c.code == rc_code)
                res = sess.execute(sel).fetchone()
                rc_id = res[0]
                rc_ids[rc_code] = rc_id
        for cap_code in prof['capabilities']:
            if cap_code not in cap_ids:
                sel = sa.select([cap_tbl.c.id]).where(
                    cap_tbl.c.code == cap_code)
                res = sess.execute(sel).fetchone()
                cap_id = res[0]
                cap_ids[cap_code] = cap_id
    ctx.status_ok()

    ctx.status("caching partition, distance type and distance internal IDs")
    for p in ctx.deployment_config.providers.values():
        # Populate our hashmap of partition name to internal ID
        part_uuid = p.partition.uuid
        if part_uuid not in part_ids:
            # Grab the partition internal ID matching the partition UUID
            sel = sa.select([part_tbl.c.id]).where(
                part_tbl.c.uuid == part_uuid)
            res = sess.execute(sel).fetchone()
            part_ids[part_uuid] = res[0]
        # Populate our hashmap of distance type and codes to internal IDs
        for pd in p.distances:
            d_key = (pd.distance_type, pd.distance_code)
            if d_key not in distance_ids:
                if pd.distance_type not in distance_type_ids:
                    # Grab the distance type internal ID matching the distance
                    # type code
                    sel = sa.select([dt_tbl.c.id]).where(
                        dt_tbl.c.code == pd.distance_type)
                    res = sess.execute(sel).fetchone()
                    distance_type_ids[pd.distance_type] = res[0]
                dt_id = distance_type_ids[pd.distance_type]
                # Not yet in cache... look up the provider group by name
                sel = sa.select([d_tbl.c.id]).where(
                    sa.and_(d_tbl.c.type_id == dt_id,
                            d_tbl.c.code == pd.distance_code))
                res = sess.execute(sel).fetchone()
                distance_ids[d_key] = res[0]
    ctx.status_ok()

    object_type_id = get_object_type_map()['runm.provider']
    compute_prov_type_id = get_provider_type_map()['runm.compute']
    ctx.status("creating providers")
    try:
        for p in ctx.deployment_config.providers.values():
            # Create the object lookup record
            obj_rec = dict(
                object_type=object_type_id,
                uuid=p.uuid,
                name=p.name,
            )
            ins = obj_tbl.insert().values(**obj_rec)
            sess.execute(ins)

            # Create the base provider record
            part_id = part_ids[p.partition.uuid]
            p_rec = dict(
                uuid=p.uuid,
                type_id=compute_prov_type_id,
                partition_id=part_id,
                generation=1,
            )
            ins = p_tbl.insert().values(**p_rec)
            res = sess.execute(ins)
            p_id = res.inserted_primary_key[0]

            # Now that we've added the base records, go ahead and flesh out
            # the provider group members and provider tree records, both of
            # which require some lookups into the provider_groups and
            # providers tables to get internal IDs.
            tree_rec = dict(
                root_provider_id=p_id,
                nested_left=1,
                nested_right=2,
                generation=1,
            )
            ins = tree_tbl.insert().values(**tree_rec)
            sess.execute(ins)

            # Add the provider's group memberships
            for pg in p.groups:
                pg_id = pg_ids[pg.uuid]
                pg_member_rec = dict(
                    provider_group_id=pg_id,
                    provider_id=p_id,
                )
                ins = pg_members_tbl.insert().values(**pg_member_rec)
                sess.execute(ins)

            # OK, now add the inventory records for the provider
            for rc_code, inv in p.profile.inventory.items():
                rc_id = rc_ids[rc_code]
                inv_rec = dict(
                    provider_id=p_id,
                    resource_class_id=rc_id,
                    total=inv['total'],
                    reserved=inv['reserved'],
                    min_unit=inv['min_unit'],
                    max_unit=inv['max_unit'],
                    step_size=inv['step_size'],
                    allocation_ratio=inv['allocation_ratio'],
                )
                ins = inv_tbl.insert().values(**inv_rec)
                sess.execute(ins)

            # Add the provider's capabilities
            for cap_code in p.profile.capabilities:
                cap_id = cap_ids[cap_code]
                p_cap_rec = dict(
                    provider_id=p_id,
                    capability_id=cap_id,
                )
                ins = p_caps_tbl.insert().values(**p_cap_rec)
                sess.execute(ins)

            # Add the distance relationships after all the provider groups have
            # been added
            for pd in p.distances:
                pg_id = pg_ids[pd.provider_group.uuid]
                d_key = (pd.distance_type, pd.distance_code)
                d_id = distance_ids[d_key]
                pd_rec = dict(
                    provider_id=p_id,
                    provider_group_id=pg_id,
                    distance_id=d_id,
                )
                ins = pd_tbl.insert().values(**pd_rec)
                sess.execute(ins)

        sess.commit()
        ctx.status_ok()
    except Exception as err:
        sess.rollback()
        ctx.status_fail(err)


def load(ctx):
    reset_db(ctx)
    create_object_types(ctx)
    create_provider_types(ctx)
    create_resource_classes(ctx)
    create_consumer_types(ctx)
    create_capabilities(ctx)
    create_distances(ctx)
    create_partitions(ctx)
    create_provider_groups(ctx)
    create_providers(ctx)
