# Loads up the runm-resource database with records that we then use in our PoC
# scenarios

import sqlalchemy as sa

import resource_models


def create_resource_classes():
    rc_tbl = resource_models.get_table('resource_classes')
    sess = resource_models.get_session()

    rcs = [
        dict(code='VCPU', description='virtual CPU'),
        dict(code='MEMORY_BYTES', description='Bytes of RAM'),
        dict(code='BLOCK_STORAGE_BYTES', description='Bytes of block storage'),
        dict(code='VGPU', description='virtual GPU context'),
    ]

    for rec in rcs:
        ins = rc_tbl.insert().values(**rec)
        sess.execute(ins)
    sess.commit()


def main():
    create_resource_classes()


if __name__ == '__main__':
    main()
