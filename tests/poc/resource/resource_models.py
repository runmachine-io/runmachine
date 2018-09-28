# Base model objects for resource database

import os
import uuid as uuidlib

import sqlalchemy as sa
from sqlalchemy.orm import sessionmaker

_TABLE_NAMES = (
    'allocation_items',
    'allocations',
    'capabilities',
    'consumer_types',
    'consumers',
    'distance_types',
    'distances',
    'inventories',
    'object_names',
    'object_types',
    'partitions',
    'provider_capabilities',
    'provider_distances',
    'provider_group_members',
    'provider_groups',
    'provider_trees',
    'provider_types',
    'providers',
    'resource_classes',
)
_TABLES = {}


def get_engine():
    db_user = os.environ.get('RUNM_TEST_RESOURCE_DB_USER', 'root')
    db_pass = os.environ.get('RUNM_TEST_RESOURCE_DB_PASS', '')
    db_uri = 'mysql+pymysql://{0}:{1}@localhost/test_resources'
    db_uri = db_uri.format(db_user, db_pass)
    return sa.create_engine(db_uri)


def get_session():
    engine = get_engine()
    sess = sessionmaker(bind=engine)
    return sess()


def load_tables():
    if _TABLES:
        return

    engine = get_engine()
    meta = sa.MetaData(engine)
    for tbl_name in _TABLE_NAMES:
        _TABLES[tbl_name] = sa.Table(tbl_name, meta, autoload=True)


def get_table(tbl_name):
    load_tables()
    return _TABLES[tbl_name]


class ProviderGroupDistance(object):
    def __init__(self, provider_group, distance_type, distance_code):
        self.provider_group = provider_group
        self.distance_type = distance_type
        self.distance_code = distance_code


class ProviderGroup(object):
    def __init__(self, name, uuid=None):
        self.name = name
        self.uuid = uuid or str(uuidlib.uuid4()).replace('-', '')
        self.distances = []

    @property
    def name_parts(self):
        name_parts = self.name.split('-')
        site_name = name_parts[0]
        row_id = None
        rack_id = None
        if len(name_parts) == 3:
            row_id = name_parts[1][len('row'):]
            rack_id = name_parts[2][len('rack'):]
        elif len(name_parts) == 2:
            row_id = name_parts[1][len('row'):]
        return site_name, row_id, rack_id

    @property
    def is_site(self):
        name_parts = self.name.split('-')
        return len(name_parts) == 1

    @property
    def is_row(self):
        name_parts = self.name.split('-')
        return len(name_parts) == 2

    @property
    def is_rack(self):
        name_parts = self.name.split('-')
        return len(name_parts) == 3

    def __repr__(self):
        return "ProviderGroup(name=%s,uuid=%s)" % (self.name, self.uuid)


class Partition(object):
    def __init__(self, name, uuid=None):
        self.name = name
        self.uuid = uuid or str(uuidlib.uuid4()).replace('-', '')

    def __repr__(self):
        return "Partition(name=%s,uuid=%s)" % (self.name, self.uuid)


class Provider(object):
    def __init__(self, name=None, partition=None, groups=None, profile=None, id=None, uuid=None):
        self.id = id
        self.name = name
        self.partition = partition
        self.uuid = uuid or str(uuidlib.uuid4()).replace('-', '')
        # Collection of provider group objects this provider is in
        self.groups = groups
        self.profile = profile

    @property
    def name_parts(self):
        name_parts = self.name.split('-')
        site_name = name_parts[0]
        row_id = None
        rack_id = None
        node_id = None
        if len(name_parts) == 4:
            row_id = name_parts[1][len('row'):]
            rack_id = name_parts[2][len('rack'):]
            node_id = name_parts[3][len('node'):]
        elif len(name_parts) == 3:
            row_id = name_parts[1][len('row'):]
            rack_id = name_parts[2][len('rack'):]
        elif len(name_parts) == 2:
            row_id = name_parts[1][len('row'):]
        return site_name, row_id, rack_id, node_id

    def __repr__(self):
        name_str = ""
        if self.name:
            name_str = ",name=" + self.name
        profile_str = ""
        if self.profile:
            profile_str = ",profile=%s" % self.profile
        return "Provider(uuid=%s%s%s)" % (
            self.uuid, name_str, profile_str)


class Consumer(object):
    def __init__(self, name, uuid=None, project=None, user=None):
        self.name = name
        self.uuid = uuid or str(uuidlib.uuid4()).replace('-', '')
        self.project = project
        self.user = user

    def __repr__(self):
        uuid_str = ""
        if self.uuid:
            uuid_str = ",uuid=" + self.uuid
        project_str = ""
        if self.project:
            project_str = ",project=" + self.project
        user_str = ""
        if self.user:
            user_str = ",user=" + self.user
        return "Consumer(name=%s%s%s%s)" % (
            self.name,
            uuid_str,
            project_str,
            user_str,
        )


class AllocationItem(object):
    def __init__(self, provider, resource_class, used):
        self.provider = provider
        self.resource_class = resource_class
        self.used = used

    def __repr__(self):
        return "\n\t\tAllocationItem(provider=%s,resource_class=%s,used=%d)" % (
            self.provider,
            self.resource_class,
            self.used,
        )


class Allocation(object):
    def __init__(self, consumer, claim_time, release_time, items):
        self.consumer = consumer
        self.claim_time = claim_time
        self.release_time = release_time
        self.items = items

    def __repr__(self):
        return "\n\tAllocation(consumer=%s,claim_time=%s,release_time=%s,items=%s)" % (
            self.consumer,
            self.claim_time,
            self.release_time,
            self.items,
        )
