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

    def _load_claim_request_groups(self, config_data):
        req_groups = []
        for request_group in config_data['request_groups']:
            res_constraints = []
            for rc_name, res_request in request_group['resources'].items():
                if 'min' not in res_request and 'max' not in res_request:
                    raise ValueError("Either min or max must be set for "
                                     "resource request group for %s" % rc_name)
                min_amount = res_request.get('min', res_request.get('max'))
                max_amount = res_request.get('max', res_request.get('min'))
                res_constraint = claim.ResourceConstraint(
                    rc_name, min_amount, max_amount)
                res_constraints.append(res_constraint)
            req_group = claim.ClaimRequestGroup(
                resource_constraints=res_constraints)
            req_groups.append(req_group)
        self.claim_request_groups = req_groups
