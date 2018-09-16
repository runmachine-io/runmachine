# Base model objects for resource database

import os
import uuid

import sqlalchemy as sa
from sqlalchemy.orm import sessionmaker

_TABLE_NAMES = (
    'object_names',
    'resource_classes',
    'capabilities',
    'distance_types',
    'consumer_types',
    'distances',
    'partitions',
    'providers',
    'provider_trees',
    'provider_capabilities',
    'provider_distances',
    'consumers',
    'provider_groups',
    'provider_group_members',
    'inventories',
    'allocations',
    'allocation_items',
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
    def __init__(self, name):
        self.name = name
        self.uuid = str(uuid.uuid4()).replace('-', '')
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
    def __init__(self, name):
        self.name = name
        self.uuid = str(uuid.uuid4()).replace('-', '')

    def __repr__(self):
        return "Partition(name=%s,uuid=%s)" % (self.name, self.uuid)


class Provider(object):
    def __init__(self, name, partition, groups, profile):
        self.name = name
        self.partition = partition
        self.uuid = str(uuid.uuid4()).replace('-', '')
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
        return "Provider(name=%s,uuid=%s,profile=%s)" % (
            self.name, self.uuid, self.profie.name)
