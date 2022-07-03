import time
from os.path import dirname
from subprocess import check_output

import pytest
import requests
from syncloudlib.integration.hosts import add_host_alias
from syncloudlib.integration.screenshots import screenshots

DIR = dirname(__file__)


@pytest.fixture(scope="session")
def module_setup(request, device, ui_mode, artifact_dir):
    def module_teardown():
        tmp_dir = '/tmp/syncloud/ui'
        device.activated()
        device.run_ssh('mkdir -p {0}'.format(tmp_dir), throw=False)
        device.run_ssh('journalctl > {0}/journalctl.ui.{1}.log'.format(tmp_dir, ui_mode), throw=False)
        device.run_ssh('cp /var/log/syslog {0}/syslog.ui.{1}.log'.format(tmp_dir, ui_mode), throw=False)
      
        device.scp_from_device('{0}/*'.format(tmp_dir), artifact_dir)
        check_output('chmod -R a+r {0}'.format(artifact_dir), shell=True)

    request.addfinalizer(module_teardown)


def test_start(module_setup, app, domain, device_host):
    add_host_alias(app, device_host, domain)


def test_login(driver, app_domain, ui_mode, screenshot_dir):
    url = "https://{0}".format(app_domain)
    driver.get(url)
    time.sleep(10)
    
    screenshots(driver, screenshot_dir, 'login-' + ui_mode)


def test_index(driver, device_user, device_password, ui_mode, screenshot_dir):
    user = driver.find_element_by_name("login")
    user.send_keys(device_user)
    password = driver.find_element_by_name("password")
    password.send_keys(device_password)
    password.submit()
    time.sleep(5)
    screenshots(driver, screenshot_dir, 'index-' + ui_mode)


def test_certificates(driver, app_domain, ui_mode, screenshot_dir):
    driver.get("https://{0}/certificates".format(app_domain))
    time.sleep(2)
    screenshots(driver, screenshot_dir, 'certificates-' + ui_mode)


def test_new_certificates(driver, ui_mode, screenshot_dir):
    name = driver.find_element_by_id("Name")
    name.send_keys("test")

    driver.find_element_by_xpath("//button[contains(text(),'Create')]").click()

    screenshots(driver, screenshot_dir, 'certificates-new-' + ui_mode)


def test_certificate(app_domain):
    response = requests.get('https://{0}'.format(app_domain), verify=False)
    assert response.status_code == 200


def test_teardown(driver):
    driver.quit()
