# Functions that simulate the claim and placement processes

import sqlalchemy as sa

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


class Allocation(object):
    def __init__(self, consumer, claim_time, release_time, items):
        self.consumer = consumer
        self.claim_time = claim_time
        self.release_time = release_time
        self.items = items

    def __repr__(self):
        return "Allocation(consumer=%s,claim_time=%s,release_time=%s,items=%s)" % (
            self.consumer,
            self.claim_time,
            self.release_time,
            self.items,
        )


def process_claim_request(ctx, claim_request):
    """Given a claim request object, ask the resource database to construct
    Claim objects that meet the request's constraints.

    :param ctx: the RunContext object
    :param claim_request: the ClaimRequest object
    """
    items = []
    alloc = Allocation(
        claim_request.consumer, claim_request.claim_time,
        claim_request.release_time, items,
    )
    item_to_group_map = {}
    return [
        Claim(alloc, item_to_group_map),
    ]
