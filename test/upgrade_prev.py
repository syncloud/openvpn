import pytest
import requests
from subprocess import check_output
from syncloudlib.integration.hosts import add_host_alias
from test import snapd

TMP_DIR = '/tmp/syncloud'
DATA_DIR = '/var/snap/openvpn/current'


@pytest.fixture(scope="session")
def module_setup(request, device, artifact_dir):
    def module_teardown():
        device.run_ssh('mkdir -p {0}'.format(TMP_DIR), throw=False)
        device.run_ssh('journalctl > {0}/upgrade-prev.journalctl.log'.format(TMP_DIR), throw=False)
        device.scp_from_device('{0}/*'.format(TMP_DIR), artifact_dir, throw=False)
        check_output('chmod -R a+r {0}'.format(artifact_dir), shell=True)

    request.addfinalizer(module_teardown)


def test_start(module_setup, device, app, domain, device_host):
    add_host_alias(app, device_host, domain)
    device.activated()
    snapd.settle(device)
    device.run_ssh('rm -rf {0}'.format(TMP_DIR), throw=False)
    device.run_ssh('mkdir -p {0}'.format(TMP_DIR), throw=False)


def test_install_released(device, app):
    device.run_ssh('snap remove {0}'.format(app), throw=False)
    device.run_ssh('snap install {0}'.format(app), retries=10)
    device.run_ssh('test -f {0}/pki/ca.crt'.format(DATA_DIR), retries=100)


def test_released_app_is_usable(app_domain):
    response = requests.get('https://{0}/auth/login'.format(app_domain),
                            verify=False, allow_redirects=False)
    assert response.status_code == 302, response.text
