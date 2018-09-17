# Functions that simulate the claim and placement processes

import sqlalchemy as sa
from sqlalchemy import func

import resource_models


class ResourceConstraint(object):
    def __init__(self, resource_class, amount):
        self.resource_class = resource_class
        self.amount = amount


class CapabilityConstraint(object):
    def __init__(self, require_caps, forbid_caps, any_caps):
        self.require_caps = require_caps
        self.forbid_caps = forbid_caps
        self.any_caps = any_caps


class ProviderGroupConstraint(object):
    def __init__(self, require_groups, forbid_groups, any_groups):
        self.require_groups = require_groups
        self.forbid_groups = forbid_groups
        self.any_groups = any_groups


class DistanceConstraint(object):
    def __init__(self, provider, minimum=None, maximum=None):
        self.provider = provider
        self.minimum = minimum
        self.maximum = maximum


class ClaimRequestGroupOptions(object):
    def __init__(self, single_provider=True, isolate_from=None):
        self.single_provider = single_provider
        self.isolate_from = isolate_from


class ClaimRequestGroup(object):
    def __init__(self, options=None, resource_constraints=None,
            capability_constraints=None, provider_group_constraints=None,
            distance_constraints=None):
        self.options = options or ClaimRequestGroupOptions()
        self.resource_constraints = resource_constraints
        self.capability_constraints = capability_constraints
        self.provider_group_constraints = provider_group_constraints
        self.distance_constraints = distance_constraints


class ClaimRequest(object):
    def __init__(self, consumer, groups, claim_time=None, release_time=None):
        self.consumer = consumer
        self.groups = groups
        self.claim_time = claim_time
        self.release_time = release_time


class Claim(object):
    def __init__(self, allocation, alloc_item_group_map):
        self.allocation = allocation
        self.allocation_item_to_request_groups = alloc_item_group_map

    def __repr__(self):
        return "Claim(allocation=%s)" % self.allocation


def process_claim_request(ctx, claim_request):
    """Given a claim request object, ask the resource database to construct
    Claim objects that meet the request's constraints.

    :param ctx: the RunContext object
    :param claim_request: the ClaimRequest object
    """
    alloc_items = []
    # A hashmap of resource class code to list of providers having capacity for
    # an amount of that resource
    rc_providers = {}
    # The set of provider internal ID, that have been matched for previous
    # iterations of constraints
    matched_provs = set()
    for rc_constraint in claim_request.groups[0].resource_constraints:
        providers = _find_providers_with_resource(
            ctx, claim_request.claim_time, claim_request.release_time,
            rc_constraint)
        if not providers:
            print "Failed to find provider with capacity for %d %s" % (
                rc_constraint.amount, rc_constraint.resource_class
            )
            return []

        print "Found %d providers with capacity for %d %s" % (
            len(providers), rc_constraint.amount, rc_constraint.resource_class
        )
        rc_provider_ids = set(p.id for p in providers)
        if matched_provs:
            matched_provs &= rc_provider_ids
            if not matched_provs:
                return []
        else:
            matched_provs = rc_provider_ids
        rc_providers.update({p.id: p for p in providers})

    # Now add an allocation item for the first provider that is in the
    # matched_provs set for each resource class in the constraint
    chosen_id = iter(matched_provs).next()
    chosen = rc_providers[chosen_id]
    for rc_constraint in claim_request.groups[0].resource_constraints:
        # Add the first provider supplying this resource class to our
        # allocation
        alloc_item = resource_models.AllocationItem(
            resource_class=rc_constraint.resource_class,
            provider=chosen,
            used=rc_constraint.amount,
        )
        alloc_items.append(alloc_item)
    alloc = resource_models.Allocation(
        claim_request.consumer, claim_request.claim_time,
        claim_request.release_time, alloc_items,
    )
    item_to_group_map = {}
    return [
        Claim(alloc, item_to_group_map),
    ]


def _rc_id_from_code(ctx, resource_class):
    rc_tbl = resource_models.get_table('resource_classes')
    sel = sa.select([rc_tbl.c.id]).where(rc_tbl.c.code == resource_class)

    sess = resource_models.get_session()
    res = sess.execute(sel).fetchone()
    if not res:
        raise ValueError("Could not find ID for resource class %s" %
                         resource_class)
    return res[0]


def _find_providers_with_resource(ctx, claim_time, release_time,
        resource_constraint):
    p_tbl = resource_models.get_table('providers')
    inv_tbl = resource_models.get_table('inventories')
    alloc_tbl = resource_models.get_table('allocations')
    alloc_item_tbl = resource_models.get_table('allocation_items')

    rc_id = _rc_id_from_code(ctx, resource_constraint.resource_class)
    alloc_window_cols = [
        alloc_tbl.c.id.label('allocation_id'),
    ]
    allocs_in_window_subq = sa.select(alloc_window_cols).where(
        sa.and_(
            alloc_tbl.c.claim_time >= claim_time,
            alloc_tbl.c.release_time < release_time,
        )
    ).group_by(alloc_tbl.c.id)
    allocs_in_window_subq = sa.alias(allocs_in_window_subq, "allocs_in_window")
    usage_cols = [
        alloc_item_tbl.c.provider_id,
        func.sum(alloc_item_tbl.c.used).label('total_used'),
    ]
    alloc_item_to_alloc_window = sa.outerjoin(
        alloc_item_tbl, allocs_in_window_subq,
        alloc_item_tbl.c.allocation_id == allocs_in_window_subq.c.allocation_id
    )
    usage_subq = sa.select(usage_cols).select_from(
        alloc_item_to_alloc_window
    ).where(
        alloc_item_tbl.c.resource_class_id == rc_id
    ).group_by(
        alloc_item_tbl.c.provider_id
    )
    usage_subq = sa.alias(usage_subq, "usages")

    p_to_inv = sa.join(
        p_tbl, inv_tbl,
        sa.and_(
            p_tbl.c.id == inv_tbl.c.provider_id,
            inv_tbl.c.resource_class_id == rc_id,
        )
    )
    inv_to_usage = sa.outerjoin(
        p_to_inv, usage_subq,
        inv_tbl.c.provider_id == usage_subq.c.provider_id
    )
    cols = [
        p_tbl.c.id,
        p_tbl.c.uuid,
    ]
    sel = sa.select(cols).select_from(
        inv_to_usage
    ).where(
        sa.and_(
            inv_tbl.c.resource_class_id == rc_id,
            ((inv_tbl.c.total - inv_tbl.c.reserved)
                * inv_tbl.c.allocation_ratio)
            >= (resource_constraint.amount + func.coalesce(usage_subq.c.total_used, 0)))
    ).limit(50)
    sess = resource_models.get_session()
    return [
        resource_models.Provider(id=r[0], uuid=r[1]) for r in sess.execute(sel)
    ]
