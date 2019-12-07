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


def test_index(driver, app_domain, device_user, device_password, ui_mode, screenshot_dir):
    user = driver.find_element_by_name("login")
    user.send_keys(device_user)
    password = driver.find_element_by_name("password")
    password.send_keys(device_password)
    password.submit()
    time.sleep(5)
    screenshots(driver, screenshot_dir, 'index-' + ui_mode)


def test_password_edit(driver, app_domain, device_user, device_password, ui_mode, screenshot_dir):
    driver.find_element_by_xpath("//a[contains(text(),'Self Modify')]").click()
   
    password = driver.find_element_by_id("password1")
    password.send_keys(device_password)
    password1 = driver.find_element_by_id("#password2")
    password1.send_keys(device_password)
    password1.submit()
    
    wait = WebDriverWait(driver, 10)
    done_message = (By.XPATH, "//span[@data-notify='message' and contains(text(),'Self modification done')]")
    wait.until(EC.presence_of_element_located(done_message))
    screenshots(driver, screenshot_dir, 'edit-' + ui_mode)
    wait.until(EC.invisibility_of_element_located(done_message))


def test_new_user(driver, app_domain, device_user, device_password, ui_mode, screenshot_dir):
    new_user_btn = "//a[contains(text(),'Add User')]"
    wait = WebDriverWait(driver, 10)
    wait.until(EC.presence_of_element_located((By.XPATH, new_user_btn)))
    screenshots(driver, screenshot_dir, 'add-user-before-open-' + ui_mode)

    driver.find_element_by_xpath(new_user_btn).click()
    time.sleep(5)
    screenshots(driver, screenshot_dir, 'add-user-after-open-' + ui_mode)

    save_btn = "//button[contains(string(),'Add User')]"
    wait.until(EC.presence_of_element_located((By.XPATH, save_btn)))
   
    driver.find_element_by_id("attr.cn").send_keys("username")
    driver.find_element_by_id("attr.name").send_keys("Last Name")
    driver.find_element_by_id("attr.first-name").send_keys("First Name")
    driver.find_element_by_name("attr.password1").send_keys("Password1!")
    driver.find_element_by_name("attr.password2").send_keys("Password1!")
    driver.find_element_by_id("attr.email").send_keys("test@example.com")

    screenshots(driver, screenshot_dir, 'add-user-before-save-' + ui_mode)

    driver.find_element_by_xpath(save_btn).click()
    time.sleep(5)
    screenshots(driver, screenshot_dir, 'add-user-saved-' + ui_mode)
    
    assert not len(driver.find_elements_by_xpath("//h4[contains(string(),'An error occured')]"))

def test_search(driver, app_domain, device_user, device_password, ui_mode, screenshot_dir):
    search_btn = "//a[contains(text(),'Delete/Modify User')]"
    driver.find_element_by_xpath(search_btn).click()
    search = driver.find_element_by_id("searchstring")
    driver.find_element_by_id("submit").click()
    wait = WebDriverWait(driver, 10)
    modify_btn = "//td/a[contains(text(),'Modify')]"
    wait.until(EC.presence_of_element_located((By.XPATH, modify_btn)))
  
    screenshots(driver, screenshot_dir, 'search-' + ui_mode)


def test_modify_user(driver, app_domain, device_user, device_password, ui_mode, screenshot_dir):
    search_btn = "//a[contains(text(),'Delete/Modify User')]"
    driver.find_element_by_xpath(search_btn).click()
    
    search = driver.find_element_by_id("searchstring")
    search.send_keys("Last Name")
    driver.find_element_by_id("submit").click()

    wait = WebDriverWait(driver, 10)
    modify_btn = "//td/a[contains(text(),'Modify')]"
    wait.until(EC.presence_of_element_located((By.XPATH, modify_btn)))
    driver.find_element_by_xpath(modify_btn).click()

    admin_role_btn = "bootstrap-switch-container"
    wait.until(EC.presence_of_element_located((By.CLASS_NAME, admin_role_btn)))

    driver.find_element_by_class_name(admin_role_btn).click()
    time.sleep(2)
    save_btn = "//button[contains(string(),'Modify User')]"
    driver.find_element_by_xpath(save_btn).click()
    time.sleep(2)
    
    screenshots(driver, screenshot_dir, 'modify-user-' + ui_mode)
    assert not len(driver.find_elements_by_xpath("//h4[contains(string(),'An error occured')]"))


def test_modify_same_user(driver, app_domain, device_user, device_password, ui_mode, screenshot_dir):
    search_btn = "//a[contains(text(),'Delete/Modify User')]"
    driver.find_element_by_xpath(search_btn).click()
    
    search = driver.find_element_by_id("searchstring")
    search.send_keys(device_user)
    driver.find_element_by_id("submit").click()
    time.sleep(2)
    
    wait = WebDriverWait(driver, 10)
    modify_btn = "//td/a[contains(text(),'Modify')]"
    wait.until(EC.presence_of_element_located((By.XPATH, modify_btn)))
    driver.find_element_by_xpath(modify_btn).click()

    name_id = "attr.name"
    wait.until(EC.presence_of_element_located((By.ID, name_id)))
    driver.find_element_by_id(name_id).send_keys("name")

    screenshots(driver, screenshot_dir, 'modify-same-user-before-' + ui_mode)

    save_btn = "//button[contains(string(),'Modify User')]"
    driver.find_element_by_xpath(save_btn).click()
    time.sleep(2)
    
    screenshots(driver, screenshot_dir, 'modify-same-user-' + ui_mode)
    assert not len(driver.find_elements_by_xpath("//h4[contains(string(),'An error occured')]"))

