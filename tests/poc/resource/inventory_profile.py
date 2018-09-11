# An inventory profile describes the providers, their inventory, traits and
# aggregate relationships for an entire load scenario

import copy
import os
import uuid

import yaml


class ProviderGroup(object):

    def __init__(self, name):
        self.name = name
        self.uuid = str(uuid.uuid4()).replace('-', '')

    def __repr__(self):
        return "ProviderGroup(name=%s,uuid=%s)" % (self.name, self.uuid)


class Provider(object):

    def __init__(self, name, groups, profile):
        self.name = name
        self.uuid = str(uuid.uuid4()).replace('-', '')
        # Collection of provider group objects this provider is in
        self.groups = groups
        self.profile = profile

    def __repr__(self):
        return "Provider(name=%s,uuid=%s,profile=%s)" % (
            self.name, self.uuid, self.profie.name)


class Profile(object):
    def __init__(self, name, inventory):
        self.name = name
        self.inventory = inventory

    def __repr__(self):
        return "Profile(name=%s)" % self.name


class InventoryProfile(object):
    def __init__(self, fp):
        """Loads the profile from a supplied filepath to a YAML file."""
        if not fp.endswith('.yaml'):
            fp = fp + '.yaml'
        if not os.path.exists(fp):
            raise RuntimeError("Unable to load inventory profile %s. "
                               "File does not exist." % fp)

        with open(fp, 'rb') as f:
            try:
                profile_dict = yaml.load(f)
            except yaml.YAMLError as err:
                raise RuntimeError("Unable to load inventory profile %s. "
                                   "Problem parsing file: %s." % (fp, err))
        self.layout = profile_dict['layout']
        self.profiles = profile_dict['profiles']
        # A hashmap of profiles by site name that compute hosts will use for
        # inventory and traits
        self.site_profiles = {}
        self._load_site_profiles()
        # A hashmap of provider group name to provider group object
        self.provider_groups = {}
        self._load_provider_groups()

    def _load_site_profiles(self):
        for prof_name, prof in self.profiles.items():
            for site_name in prof['sites']:
                prof_inv = {}
                for rc_name, inv in prof['inventory'].items():
                    if 'min_unit' not in inv:
                        inv['min_unit'] = 1
                    if 'max_unit' not in inv:
                        inv['max_unit'] = inv['total']
                    if 'step_size' not in inv:
                        inv['step_size'] = 1
                    if 'allocation_ratio' not in inv:
                        inv['allocation_ratio'] = 1.0
                    if 'reserved' not in inv:
                        inv['reserved'] = 0
                    prof_inv[rc_name] = inv
                self.site_profiles[site_name] = Profile(prof_name, prof_inv)

    def _load_provider_groups(self):
        for site_name in self.layout['sites']:
            pg = ProviderGroup(site_name)
            self.provider_groups[pg.name] = pg

            for row_id in range(self.count_rows_per_site):
                pg_name = "%s-row%s" % (
                        site_name,
                        row_id,
                    )
                pg = ProviderGroup(pg_name)
                self.provider_groups[pg.name] = pg
                for rack_id in range(self.count_racks_per_row):
                    pg_name = "%s-row%s-rack%s" % (
                            site_name,
                            row_id,
                            rack_id,
                        )
                    pg = ProviderGroup(pg_name)
                    self.provider_groups[pg.name] = pg

    @property
    def count_rows_per_site(self):
        return int(self.layout.get('rows_per_site', 0))

    @property
    def count_racks_per_row(self):
        return int(self.layout.get('racks_per_row', 0))

    @property
    def count_nodes_per_rack(self):
        return int(self.layout.get('nodes_per_rack', 0))

    @property
    def iter_providers(self):
        """Yields instructions for creating a provider, its inventory, traits
        and group associations.
        """
        for site_name in self.layout['sites']:
            for row_id in range(self.count_rows_per_site):
                for rack_id in range(self.count_racks_per_row):
                    for node_id in range(self.count_nodes_per_rack):
                        pg_names = [
                            site_name,
                            "%s-row%s" % (site_name, row_id),
                            "%s-row%s-rack%s" % (site_name, row_id, rack_id),
                        ]
                        provider_name = "%s-row%s-rack%s-node%s" % (
                            site_name,
                            row_id,
                            rack_id,
                            node_id,
                        )
                        groups = [
                            self.provider_groups[pg_name]
                            for pg_name in pg_names
                        ]
                        profile = self.site_profiles[site_name]
                        yield Provider(provider_name, groups, profile)
