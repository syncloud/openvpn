from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.common.by import By


def login(selenium, device_user, device_password):
    selenium.open_app()
    selenium.screenshot('index')
    user = selenium.find_by_name("login")
    user.send_keys(device_user)
    password = selenium.find_by_name("password")
    password.send_keys(device_password)
    password.submit()
    
    selenium.screenshot('index')                        

