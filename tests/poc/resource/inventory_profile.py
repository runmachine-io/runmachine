# An inventory profile describes the providers, their inventory, traits and
# aggregate relationships for an entire load scenario

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

    def __init__(self, name, groups):
        self.name = name
        self.uuid = str(uuid.uuid4()).replace('-', '')
        # Collection of provider group objects this provider is in
        self.groups = groups

    def __repr__(self):
        return "Provider(name=%s,uuid=%s)" % (self.name, self.uuid)


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
        # A hashmap of provider group name to provider group object
        self.provider_groups = {}
        self._load_from_dict(profile_dict)
        self._load_provider_groups()

    def _load_from_dict(self, profile_dict):
        self.compute = profile_dict['compute']
        self.site_names = profile_dict.get('site_names')

    def _load_provider_groups(self):
        for site_id in range(self.count_sites):
            site_name = site_id
            if self.site_names:
                site_name = self.site_names[site_id]

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
    def count_sites(self):
        return int(self.compute.get('sites', 0))

    @property
    def count_rows_per_site(self):
        return int(self.compute.get('rows_per_site', 0))

    @property
    def count_racks_per_row(self):
        return int(self.compute.get('racks_per_row', 0))

    @property
    def count_nodes_per_rack(self):
        return int(self.compute.get('nodes_per_rack', 0))

    @property
    def iter_providers(self):
        """Yields instructions for creating a provider, its inventory, traits
        and group associations.
        """
        for site_id in range(self.count_sites):
            site_name = site_id
            if self.site_names:
                site_name = self.site_names[site_id]

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
                        yield Provider(provider_name, groups)
