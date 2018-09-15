# An inventory profile describes the providers, their inventory, traits and
# aggregate relationships for an entire load scenario

import copy
import os
import uuid

import yaml


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
        # A hashmap of partition name to partition object
        self.partitions = {}
        self._load_partitions()
        # A hashmap of profiles by site name that compute hosts will use for
        # inventory and traits
        self.site_profiles = {}
        self._load_site_profiles()
        # A hashmap of provider group name to provider group object
        self.provider_groups = {}
        self._load_provider_groups()
        # A hashmap of provider name to provider object
        self.providers = {}
        self._load_providers()

    def _load_partitions(self):
        # For now, just have a single hard-coded partition
        self.partitions['part0'] = Partition('part0')

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

    def _calculate_distances(self, p):
        distances = []
        parts = p.name_parts
        p_site_name, p_row_id, p_rack_id, p_node_id = parts

        for pg in self.provider_groups.values():
            parts = pg.name_parts
            pg_site_name, pg_row_id, pg_rack_id = parts
            if pg_site_name != p_site_name:
                # The greatest distance is between sites. Because all nodes in
                # a different site will be in the provider group representing
                # the other site, we only need to add a distance record from
                # this provider to the provider group representing that other
                # site, not any provider group representing smaller subgroups
                # (such as a row or rack) that is IN that other site....
                if pg.is_site:
                    d = ProviderGroupDistance(
                        pg, "network", "remote")
                    distances.append(d)
                    d = ProviderGroupDistance(
                        pg, "failure", "site")
                    distances.append(d)
            else:
                # For failure domain distance, we store more fine-grained
                # distances versus network latency. For failure domains, look
                # to see if the provider group is a rack, a row or a site and
                # add the appropriate failure domain distance depending on
                # whether the provider has a matching rack or row.
                if pg.is_rack:
                    if p_row_id == pg_row_id:
                        if p_rack_id == pg_rack_id:
                            # Provider is in the same rack as the provider
                            # group so is in a "local" failure domain
                            d = ProviderGroupDistance(
                                pg, "failure", "local")
                            distances.append(d)
                            # And is in a "local" distance for network latency
                            d = ProviderGroupDistance(
                                pg, "network", "local")
                            distances.append(d)
                elif pg.is_row:
                    # There is no real difference between different racks in a
                    # row and different rows from a network latency
                    # perspective, so we add distance records representing
                    # "site-local" latency between the provider and the *row*
                    # provider group
                    d = ProviderGroupDistance(
                        pg, "network", "site")
                    distances.append(d)
                    # For failure domain, it *does* matter whether the provider
                    # is in the same row as the row provider group, since
                    # failure domains are finer-grained than network latency
                    # distances
                    if pg_row_id == p_row_id:
                        # Provider is in the same row as the row provider group
                        # so is in a different "rack" failure domain from other
                        # nodes in the row that are not also in the same rack
                        d = ProviderGroupDistance(
                            pg, "failure", "rack")
                        distances.append(d)
                    else:
                        d = ProviderGroupDistance(
                            pg, "failure", "row")
                        distances.append(d)

        p.distances = distances

    @property
    def count_rows_per_site(self):
        return int(self.layout.get('rows_per_site', 0))

    @property
    def count_racks_per_row(self):
        return int(self.layout.get('racks_per_row', 0))

    @property
    def count_nodes_per_rack(self):
        return int(self.layout.get('nodes_per_rack', 0))

    def _load_providers(self):
        """Yields instructions for creating a provider, its inventory, traits
        and group associations.
        """
        # TODO(jaypipes): Support more than a single partition in the
        # deployment config layout section
        partition = self.partitions['part0']
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

                        # OK, now we construct the distance matrix. For now,
                        # we're just going to hard-code the network latency
                        # distances between sites, rows and racks
                        p = Provider(provider_name, partition, groups, profile)
                        self._calculate_distances(p)
                        self.providers[p.name] = p
