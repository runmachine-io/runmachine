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
    def __init__(self, groups):
        self.groups = groups
