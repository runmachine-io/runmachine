# An inventory profile describes the providers, their inventory, traits and
# aggregate relationships for an entire load scenario

import os

import yaml

import claim


class ClaimConfig(object):
    def __init__(self, fp):
        """Loads the claim configuration from a supplied filepath to a YAML
        file.
        """
        if not fp.endswith('.yaml'):
            fp = fp + '.yaml'
        if not os.path.exists(fp):
            raise RuntimeError("Unable to load claim configuration %s. "
                               "File does not exist." % fp)

        with open(fp, 'rb') as f:
            try:
                config_dict = yaml.load(f)
            except yaml.YAMLError as err:
                raise RuntimeError("Unable to load claim configuration "
                                   "%s. Problem parsing file: %s." % (fp, err))
        self._load_claim_request_groups(config_dict)

    def _process_capability_constraints(self, block):
        constraints = []
        for caps in block:
            constraints.append(self._process_capability_constraint(caps))
        return constraints

    def _process_capability_constraint(self, block):
        # There are three kinds of capability constraints.
        #
        # A list of capability names under a 'require' key indicates the
        # AND'd set of required capabilities that the set of providers
        # meeting the resource constraints must have.
        #
        # A list of capability names under a 'forbid' key indicates the
        # capabilities that must not be present in any of the providers
        # meeting the resource constraints.
        #
        # A list of capability names under an 'any' key indicates the
        # providers meeting the resource constraints must have at least one
        # of the list of capabilities.
        require_caps = block.get('require')
        forbid_caps = block.get('forbid')
        any_caps = block.get('any')
        constraint = claim.CapabilityConstraint(
            require_caps=require_caps,
            forbid_caps=forbid_caps,
            any_caps=any_caps)
        return constraint

    def _process_resource_constraints(self, block):
        constraints = []
        for rc_name, res_request in block.items():
            if 'min' not in res_request and 'max' not in res_request:
                raise ValueError("Either min or max must be set for "
                                 "resource request group for %s" % rc_name)
            min_amount = res_request.get('min', res_request.get('max'))
            max_amount = res_request.get('max', res_request.get('min'))

            # Optional resource-specific capability constraint may be
            # associated with this resource
            cap_constraint = None
            if 'capabilities' in res_request:
                cap_constraint = self._process_capability_constraint(
                    res_request['capabilities'])

            constraint = claim.ResourceConstraint(
                rc_name, min_amount, max_amount,
                capability_constraint=cap_constraint)
            constraints.append(constraint)
        return constraints

    def _load_claim_request_groups(self, config_data):
        req_groups = []
        for request_group in config_data['request_groups']:
            # Handle the resource constraints, which are defined in a
            # 'resources' key in the request_group object
            res_constraints = self._process_resource_constraints(
                request_group['resources'])

            # Capabilities constraints that are under the 'capabilities' key of
            # the request group object apply to all the providers that match
            # for the group
            cap_constraints = None
            if 'capabilities' in request_group:
                cap_constraints = self._process_capability_constraints(
                    request_group['capabilities'])

            req_group = claim.ClaimRequestGroup(
                resource_constraints=res_constraints,
                capability_constraints=cap_constraints)
            req_groups.append(req_group)

        self.claim_request_groups = req_groups
