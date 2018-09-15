# Loads up the runm-resource database with records that we then use in our PoC
# scenarios

import argparse
import datetime
import os
import sys
import time

import claim
import load
import deployment_config

_LOG_FORMAT = "%(level)s %(message)s"
_DEPLOYMENT_CONFIGS_DIR = os.path.join(
    os.path.abspath(os.path.dirname(__file__)), 'deployment-configs',
)
_DEFAULT_DEPLOYMENT_CONFIG = '1k-shared-compute'


class RunContext(object):
    def __init__(self, args):
        self.args = args
        self.deployment_config = None

    def status(self, msg):
        sys.stdout.write(msg + " ... ")
        sys.stdout.flush()

    def status_ok(self, ):
        sys.stdout.write("ok\n")
        sys.stdout.flush()

    def status_fail(self, err):
        sys.stdout.write("FAIL\n")
        sys.stdout.flush()
        sys.stderr.write(" error: %s\n" % err)
        sys.stderr.flush()


def find_claims(ctx):
    consumer = "instance0"
    resource_constraints = [
        claim.ResourceConstraint("runm.cpu.shared", 2),
        claim.ResourceConstraint("runm.memory", 128*1000*1000),
    ]
    crg0 = claim.ClaimRequestGroup(resource_constraints=resource_constraints)
    request_groups = [
        crg0,
    ]
    claim_time = datetime.datetime.utcnow()
    claim_time = int(time.mktime(claim_time.timetuple()))
    release_time = sys.maxint
    cr = claim.ClaimRequest(consumer, request_groups, claim_time, release_time)
    claims = claim.process_claim_request(ctx, cr)
    for c in claims:
        print c


def setup_opts(parser):
    deployment_configs = []
    for fn in os.listdir(_DEPLOYMENT_CONFIGS_DIR):
        fp = os.path.join(_DEPLOYMENT_CONFIGS_DIR, fn)
        if os.path.isfile(fp) and fn.endswith('.yaml'):
            deployment_configs.append(fn[0:len(fn) - 5])

    parser.add_argument('--reset', action='store_true',
                        default=False, help="Do NOT reset the database.")
    parser.add_argument('--deployment-config',
                        choices=deployment_configs,
                        default=_DEFAULT_DEPLOYMENT_CONFIG,
                        help="Deployment configuration to use.")


def main(ctx):
    if ctx.args.reset:
        fp = os.path.join(_DEPLOYMENT_CONFIGS_DIR, args.deployment_config)
        ctx.deployment_config = deployment_config.DeploymentConfig(fp)
        load.reset_db(ctx)
        load.create_resource_classes(ctx)
        load.create_consumer_types(ctx)
        load.create_capabilities(ctx)
        load.create_distances(ctx)
        load.create_provider_groups(ctx)
        load.create_providers(ctx)
    find_claims(ctx)


if __name__ == '__main__':
    p = argparse.ArgumentParser(description='Load up resource database.')
    setup_opts(p)
    args = p.parse_args()
    ctx = RunContext(args)
    main(ctx)
