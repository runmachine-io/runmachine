# An inventory profile describes the providers, their inventory, traits and
# aggregate relationships for an entire load scenario

import copy
import os
import uuid

import yaml


class ProviderGroupDistance(object):
    def __init__(self, right_provider_group, distance_type, distance_code):
        self.right_provider_group = right_provider_group
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
    def __init__(self, name, inventory, capabilities):
        self.name = name
        self.inventory = inventory
        self.capabilities = capabilities

    def __repr__(self):
        return "Profile(name=%s)" % self.name


class DeploymentConfig(object):
    def __init__(self, fp):
        """Loads the deployment configuration from a supplied filepath to a
        YAML file.
        """
        if not fp.endswith('.yaml'):
            fp = fp + '.yaml'
        if not os.path.exists(fp):
            raise RuntimeError("Unable to load deployment configuration %s. "
                               "File does not exist." % fp)

        with open(fp, 'rb') as f:
            try:
                config_dict = yaml.load(f)
            except yaml.YAMLError as err:
                raise RuntimeError("Unable to load deployment configuration "
                                   "%s. Problem parsing file: %s." % (fp, err))
        self.layout = config_dict['layout']
        self.profiles = config_dict['profiles']
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
                caps = prof['capabilities']
                p = Profile(prof_name, prof_inv, caps)
                self.site_profiles[site_name] = p

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

        # OK, now we construct the distance matrix. For now, we're just going
        # to hard-code the network latency distances between sites, rows and
        # racks
        for pg in self.provider_groups.values():
            self._calculate_distances(pg)

    def _calculate_distances(self, left_pg):
        distances = []
        parts = left_pg.name_parts
        left_site_name, left_row_id, left_rack_id, left_node_id = parts

        for right_pg in self.provider_groups.values():
            if left_pg.name == right_pg.name:
                # Distance between same objects doesn't make sense
                continue
            parts = right_pg.name_parts
            right_site_name, right_row_id, right_rack_id, right_node_id = parts
            if right_site_name != left_site_name:
                # The greatest distance is between sites
                d = ProviderGroupDistance(
                    right_pg, "network", "remote")
                distances.append(d)
            else:
                # Same site, so provider groups have datacenter distance
                # between them
                d = ProviderGroupDistance(
                    right_pg, "network", "datacenter")
                distances.append(d)
        left_pg.distances = distances

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
