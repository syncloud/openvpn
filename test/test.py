import os
import pytest
import requests
from os.path import join
from subprocess import check_output
from retrying import retry
from syncloudlib.integration.hosts import add_host_alias
from syncloudlib.integration.installer import local_install

TMP_DIR = '/tmp/syncloud'

DATA_DIR = '/var/snap/openvpn/current'
APP_DIR = '/snap/openvpn/current'


@pytest.fixture(scope="session")
def module_setup(request, device, data_dir, platform_data_dir, app_dir, artifact_dir, snap_data_dir):
    def module_teardown():
        platform_log_dir = join(artifact_dir, 'platform_log')
        os.mkdir(platform_log_dir)
        device.scp_from_device('{0}/log/*'.format(platform_data_dir), platform_log_dir, throw=False)
        device.run_ssh('mkdir -p {0}'.format(TMP_DIR), throw=False)
        device.run_ssh('journalctl > {0}/journalctl.log'.format(TMP_DIR), throw=False)
        device.run_ssh('snap logs openvpn -n 500 > {0}/snap.openvpn.log'.format(TMP_DIR), throw=False)
        device.run_ssh('ps auxfw > {0}/ps.log'.format(TMP_DIR), throw=False)
        device.run_ssh('netstat -nlp > {0}/netstat.log'.format(TMP_DIR), throw=False)
        device.run_ssh('ls -la {0} > {1}/data.ls.log'.format(snap_data_dir, TMP_DIR), throw=False)
        device.run_ssh('ls -la {0}/pki > {1}/pki.ls.log'.format(snap_data_dir, TMP_DIR), throw=False)
        device.run_ssh('cat {0}/openvpn/server.conf > {1}/server.conf'.format(snap_data_dir, TMP_DIR),
                       throw=False)
        device.run_ssh('nft list table ip nat > {0}/nft.log'.format(TMP_DIR), throw=False)
        device.run_ssh('iptables -t nat -S > {0}/iptables.log'.format(TMP_DIR), throw=False)
        device.scp_from_device('{0}/*'.format(TMP_DIR), artifact_dir)
        check_output('chmod -R a+r {0}'.format(artifact_dir), shell=True)

    request.addfinalizer(module_teardown)


def test_start(module_setup, device, app, domain, device_host):
    add_host_alias(app, device_host, domain)
    device.run_ssh('date', retries=100, throw=True)
    hold_refresh(device)


def hold_refresh(device):
    device.run_ssh('snap wait system seed.loaded', retries=100, throw=False)
    device.run_ssh('snap set system refresh.hold=2099-01-01T00:00:00Z', retries=20, throw=False)


@retry(stop_max_attempt_number=30, wait_fixed=10000)
def activate(device):
    response = device.activate_custom()
    assert response.status_code == 200, response.text


def test_activate_device(device):
    activate(device)


def test_install(app_archive_path, device_host, device_session, device_password):
    local_install(device_host, device_password, app_archive_path)


def test_index(app_domain):
    response = requests.get('https://{0}'.format(app_domain), verify=False, allow_redirects=False)
    assert response.status_code in (200, 302), response.text


def test_backend_ready(device):
    device.run_ssh('test -S {0}/backend.sock'.format(DATA_DIR), retries=100)


def test_pki_generated(device):
    device.run_ssh('test -f {0}/pki/ca.crt'.format(DATA_DIR), retries=100)
    device.run_ssh('test -f {0}/pki/private/ca.key'.format(DATA_DIR))
    device.run_ssh('test -f {0}/pki/issued/server.crt'.format(DATA_DIR))
    device.run_ssh('test -f {0}/pki/crl.pem'.format(DATA_DIR))


def test_server_conf_generated(device):
    device.run_ssh('test -f {0}/openvpn/server.conf'.format(DATA_DIR), retries=100)
    conf = device.run_ssh('cat {0}/openvpn/server.conf'.format(DATA_DIR))
    assert 'topology subnet' in conf
    assert 'data-ciphers' in conf
    assert 'crl-verify' in conf


def test_server_conf_is_openvpn27_clean(device):
    conf = device.run_ssh('cat {0}/openvpn/server.conf'.format(DATA_DIR))
    assert 'comp-lzo' not in conf
    assert 'compress migrate' in conf
    assert '\ncipher ' not in conf
    assert '\ndh ' not in conf


def test_openvpn_version(device):
    version = device.run_ssh('{0}/openvpn/sbin/openvpn.sh --version | head -1'.format(APP_DIR))
    assert 'OpenVPN 2.7' in version


def test_server_running(device):
    device.run_ssh('test -S {0}/openvpn.socket'.format(DATA_DIR), retries=100)


def test_rules(device):
    rules = device.run_ssh(
        "sh -c '"
        "if command -v nft >/dev/null 2>&1; "
        "then nft list chain ip nat POSTROUTING | grep masquerade | wc -l; "
        "else iptables -t nat -S POSTROUTING | grep MASQUERADE | wc -l; fi'")
    assert 1 == int(rules.strip())


def test_prefix_delegation(device):
    device.run_ssh('{0}/bin/prefix_delegation.sh'.format(APP_DIR),
                   env_vars='reason=BOUND6 new_ip6_prefix=111:222:333::/64')
    conf = device.run_ssh('cat {0}/openvpn/server.conf'.format(DATA_DIR))
    assert 'server-ipv6 111:222:333::/64' in conf


def test_ipv6_prefix_survives_rerender(device):
    device.run_ssh('snap restart openvpn.backend')
    device.run_ssh("grep 'server-ipv6 111:222:333::/64' {0}/openvpn/server.conf".format(DATA_DIR),
                   retries=30)


def test_upgrade_same_version(app_archive_path, device_host, device_password):
    local_install(device_host, device_password, app_archive_path)


def test_remove(device, app):
    response = device.app_remove(app)
    assert response.status_code == 200, response.text


def test_reinstall(app_archive_path, device_host, device_password):
    local_install(device_host, device_password, app_archive_path)
