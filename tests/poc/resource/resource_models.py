# Base model objects for resource database

import os

import sqlalchemy as sa
from sqlalchemy.orm import sessionmaker

_TABLE_NAMES = (
    'object_names',
    'resource_classes',
    'capabilities',
    'distance_types',
    'consumer_types',
    'distances',
    'providers',
    'provider_trees',
    'provider_capabilities',
    'consumers',
    'provider_groups',
    'provider_group_members',
    'provider_group_distances',
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
