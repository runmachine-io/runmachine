# Loads up the runm-resource database with records that we then use in our PoC
# scenarios

import argparse
import os
import subprocess
import sys

import sqlalchemy as sa

import inventory_profile
import resource_models

_LOG_FORMAT = "%(level)s %(message)s"
_RESOURCE_SCHEMA_FILE = os.path.join(
    os.path.abspath(os.path.dirname(__file__)), 'resource_schema.sql',
)
_INVENTORY_PROFILES_DIR = os.path.join(
    os.path.abspath(os.path.dirname(__file__)), 'inventory-profiles',
)
_DEFAULT_INVENTORY_PROFILE = '1k-shared-compute'


class RunContext(object):
    def __init__(self, args):
        self.args = args
        self.inventory_profile = None

    def status(self, msg):
        sys.stdout.write(msg + " ... ")
        sys.stdout.flush()

    def status_ok(self, ):
        sys.stdout.write("ok\n")
        sys.stdout.flush()

    def status_fail(self, err):
        sys.stdout.write("FAIL\n")
        sys.stdout.flush()
        sys.stderr.write(" error: %s\n" % err)
        sys.stderr.flush()


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

    recs = [
        dict(
            type_id=net_dt_id,
            code="local",
            position=0,
            description="Virtually no network latency",
        ),
        dict(
            type_id=net_dt_id,
            code="datacenter",
            position=1,
            description="latency between leaf switch-connected nodes in a DC",
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
    ]
    try:
        _insert_records(d_tbl, recs)
        ctx.status_ok()
    except Exception as err:
        ctx.status_fail(err)


def create_provider_groups(ctx):
    ctx.status("creating provider groups")
    obj_tbl = resource_models.get_table('object_names')
    pg_tbl = resource_models.get_table('provider_groups')

    obj_recs = []
    pg_recs = []

    for pg in ctx.inventory_profile.provider_groups.values():
        obj_rec = dict(
            object_type='provider_group',
            uuid=pg.uuid,
            name=pg.name,
        )
        obj_recs.append(obj_rec)
        pg_rec = dict(
            uuid=pg.uuid,
        )
        pg_recs.append(pg_rec)

    try:
        _insert_records(obj_tbl, obj_recs)
        _insert_records(pg_tbl, pg_recs)
        ctx.status_ok()
    except Exception as err:
        ctx.status_fail(err)


def create_providers(ctx):
    ctx.status("creating providers")
    obj_tbl = resource_models.get_table('object_names')
    rc_tbl = resource_models.get_table('resource_classes')
    pg_tbl = resource_models.get_table('provider_groups')
    pg_members_tbl = resource_models.get_table('provider_group_members')
    p_tbl = resource_models.get_table('providers')
    tree_tbl = resource_models.get_table('provider_trees')
    inv_tbl = resource_models.get_table('inventories')

    # in-process cache of provider group name -> internal ID
    pg_ids = {}
    # in-process cache of resource class name -> internal ID
    rc_ids = {}

    sess = resource_models.get_session()
    for pg in ctx.inventory_profile.provider_groups.values():
        if pg.uuid in pg_ids:
            pg_id = pg_ids[pg.uuid]
        else:
            # Not yet in cache... look up the provider group by name
            sel = sa.select([pg_tbl.c.id]).where(pg_tbl.c.uuid == pg.uuid)
            res = sess.execute(sel).fetchone()
            pg_ids[pg.uuid] = res[0]

    for prof in ctx.inventory_profile.profiles.values():
        for rc_name in prof['inventory'].keys():
            if rc_name not in rc_ids:
                sel = sa.select([rc_tbl.c.id]).where(rc_tbl.c.code == rc_name)
                res = sess.execute(sel).fetchone()
                rc_id = res[0]
                rc_ids[rc_name] = rc_id

    try:
        for p in ctx.inventory_profile.iter_providers:
            # Create the object lookup record
            obj_rec = dict(
                object_type='provider',
                uuid=p.uuid,
                name=p.name,
            )
            ins = obj_tbl.insert().values(**obj_rec)
            sess.execute(ins)

            # Create the base provider record
            p_rec = dict(
                uuid=p.uuid,
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

            for pg in p.groups:
                pg_id = pg_ids[pg.uuid]
                pg_member_rec = dict(
                    provider_group_id=pg_id,
                    provider_id=p_id,
                )
                ins = pg_members_tbl.insert().values(**pg_member_rec)
                sess.execute(ins)

            # OK, now add the inventory records for the provider
            for rc_name, inv in p.profile.inventory.items():
                rc_id = rc_ids[rc_name]
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

        sess.commit()
        ctx.status_ok()
    except Exception as err:
        sess.rollback()
        ctx.status_fail(err)


def setup_opts(parser):
    inventory_profiles = []
    for fn in os.listdir(_INVENTORY_PROFILES_DIR):
        fp = os.path.join(_INVENTORY_PROFILES_DIR, fn)
        if os.path.isfile(fp) and fn.endswith('.yaml'):
            inventory_profiles.append(fn[0:len(fn) - 5])

    parser.add_argument('--reset', action='store_true',
                        default=True, help="Reset the database entirely.")
    parser.add_argument('--inventory-profile',
                        choices=inventory_profiles,
                        default=_DEFAULT_INVENTORY_PROFILE,
                        help="Inventory profile to use.")


def main(ctx):
    if ctx.args.reset:
        reset_db(ctx)
        create_resource_classes(ctx)
        create_consumer_types(ctx)
        create_capabilities(ctx)
        create_distances(ctx)

    fp = os.path.join(_INVENTORY_PROFILES_DIR, args.inventory_profile)
    ctx.inventory_profile = inventory_profile.InventoryProfile(fp)
    create_provider_groups(ctx)
    create_providers(ctx)


if __name__ == '__main__':
    p = argparse.ArgumentParser(description='Load up resource database.')
    setup_opts(p)
    args = p.parse_args()
    ctx = RunContext(args)
    main(ctx)
