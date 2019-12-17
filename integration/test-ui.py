import os
import shutil
import time
import pytest
from os.path import dirname, join, exists

from syncloudlib.integration.hosts import add_host_alias_by_ip
from syncloudlib.integration.screenshots import screenshots
from selenium.webdriver.common.by import By
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.ui import WebDriverWait

DIR = dirname(__file__)


@pytest.fixture(scope="session")
def module_setup(request, device, log_dir, ui_mode, artifact_dir):
    def module_teardown():
        tmp_dir = '/tmp/syncloud/ui'
        device.activated()
        device.run_ssh('mkdir -p {0}'.format(tmp_dir), throw=False)
        device.run_ssh('journalctl > {0}/journalctl.ui.{1}.log'.format(tmp_dir, ui_mode), throw=False)
        device.run_ssh('cp /var/log/syslog {0}/syslog.ui.{1}.log'.format(tmp_dir, ui_mode), throw=False)
      
        device.scp_from_device('{0}/*'.format(tmp_dir), artifact_dir)
    request.addfinalizer(module_teardown)

def test_start(module_setup, app, domain, device_host):
    add_host_alias_by_ip(app, domain, device_host)


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
