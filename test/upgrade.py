import pytest
import requests
from subprocess import check_output
from syncloudlib.integration.hosts import add_host_alias
from syncloudlib.integration.installer import local_install
from test import snapd

TMP_DIR = '/tmp/syncloud'

DATA_DIR = '/var/snap/openvpn/current'
APP_DIR = '/snap/openvpn/current'
UPGRADE_CLIENT = 'upgrade-test'


@pytest.fixture(scope="session")
def module_setup(request, device, artifact_dir):
    def module_teardown():
        device.run_ssh('mkdir -p {0}'.format(TMP_DIR), throw=False)
        device.run_ssh('journalctl > {0}/upgrade.journalctl.log'.format(TMP_DIR), throw=False)
        device.run_ssh('snap logs openvpn -n 500 > {0}/upgrade.snap.log'.format(TMP_DIR), throw=False)
        device.run_ssh('cat {0}/openvpn/server.conf > {1}/upgrade.server.conf'.format(DATA_DIR, TMP_DIR),
                       throw=False)
        device.run_ssh('cat {0}/pki/index.txt > {1}/upgrade.index.txt'.format(DATA_DIR, TMP_DIR),
                       throw=False)
        device.scp_from_device('{0}/*'.format(TMP_DIR), artifact_dir)
        check_output('chmod -R a+r {0}'.format(artifact_dir), shell=True)

    request.addfinalizer(module_teardown)


@pytest.fixture(scope="session")
def released_state(device):
    return {}


def test_start(module_setup, device, app, domain, device_host):
    add_host_alias(app, device_host, domain)
    snapd.settle(device)
    device.activated()
    device.run_ssh('rm -rf {0}'.format(TMP_DIR), throw=False)
    device.run_ssh('mkdir -p {0}'.format(TMP_DIR), throw=False)


def test_install_released(device, app):
    device.run_ssh('snap remove {0}'.format(app), throw=False)
    device.run_ssh('snap install {0}'.format(app), retries=10)
    device.run_ssh('test -f {0}/pki/ca.crt'.format(DATA_DIR), retries=100)


def test_released_creates_client(device, released_state):
    device.run_ssh(
        '{0}/easyrsa/easyrsa --vars={0}/config/easyrsa/vars --batch build-client-full {1} nopass'.format(
            APP_DIR, UPGRADE_CLIENT),
        retries=5)
    device.run_ssh('test -f {0}/pki/issued/{1}.crt'.format(DATA_DIR, UPGRADE_CLIENT))


def test_capture_released_state(device, released_state):
    released_state['ca'] = device.run_ssh(
        "sha256sum {0}/pki/ca.crt | cut -d' ' -f1".format(DATA_DIR)).strip()
    released_state['client'] = device.run_ssh(
        "sha256sum {0}/pki/issued/{1}.crt | cut -d' ' -f1".format(DATA_DIR, UPGRADE_CLIENT)).strip()
    released_state['port'] = device.run_ssh(
        "grep '^port ' {0}/openvpn/server.conf | cut -d' ' -f2".format(DATA_DIR)).strip()

    assert released_state['ca']
    assert released_state['port']


def test_upgrade(app_archive_path, device_host, device_password):
    local_install(device_host, device_password, app_archive_path)


def test_ca_is_not_regenerated(device, released_state):
    after = device.run_ssh("sha256sum {0}/pki/ca.crt | cut -d' ' -f1".format(DATA_DIR)).strip()
    assert after == released_state['ca'], \
        'the CA changed on upgrade, every already-downloaded client profile is now invalid'


def test_existing_client_cert_survives(device, released_state):
    after = device.run_ssh(
        "sha256sum {0}/pki/issued/{1}.crt | cut -d' ' -f1".format(DATA_DIR, UPGRADE_CLIENT)).strip()
    assert after == released_state['client']


def test_existing_client_still_in_index(device):
    index = device.run_ssh('cat {0}/pki/index.txt'.format(DATA_DIR))
    assert UPGRADE_CLIENT in index


def test_port_is_preserved(device, released_state):
    settings = device.run_ssh('cat {0}/settings.json'.format(DATA_DIR))
    assert '"port": {0}'.format(released_state['port']) in settings, settings

    conf = device.run_ssh('cat {0}/openvpn/server.conf'.format(DATA_DIR))
    assert 'port {0}'.format(released_state['port']) in conf


def test_upgraded_server_conf_is_27_clean(device):
    conf = device.run_ssh('cat {0}/openvpn/server.conf'.format(DATA_DIR))
    assert 'comp-lzo' not in conf
    assert 'compress migrate' in conf, \
        'legacy clients carry comp-lzo, migrate mode is what keeps them connecting'


def test_server_running_after_upgrade(device):
    device.run_ssh('test -S {0}/openvpn.socket'.format(DATA_DIR), retries=30)


def test_index_after_upgrade(app_domain):
    response = requests.get('https://{0}'.format(app_domain), verify=False, allow_redirects=False)
    assert response.status_code in (200, 302), response.text
