import logging
import os
from os.path import join, isfile
from subprocess import check_output
import shutil

from syncloudlib import fs, linux, gen, logger
from syncloudlib.application import paths, storage, urls

APP_NAME = 'openvpn'

USER_NAME = APP_NAME
DB_NAME = APP_NAME
DB_USER = APP_NAME
DB_PASSWORD = APP_NAME


class Installer:
    def __init__(self):
        if not logger.factory_instance:
            logger.init(logging.DEBUG, True)

        self.log = logger.get_logger('{0}_installer'.format(APP_NAME))
        self.app_dir = paths.get_app_dir(APP_NAME)
        self.snap_common = os.environ['SNAP_COMMON']
        self.snap_data = join('/var/snap', APP_NAME, 'current')
        self.config_path = join(self.snap_data, 'config')
        self.openvpn_config_dir = join(self.snap_data, 'openvpn')
        self.openssl_bin = join(self.app_dir, 'openssl/bin/openssl')
        self.generate_keys_bin = join(self.app_dir, 'bin/generate-keys.sh')
        self.device_domain_name = urls.get_device_domain_name()

    def install_config(self):

        home_folder = join('/home', USER_NAME)
        linux.useradd(USER_NAME, home_folder=home_folder)

        fs.makepath(join(self.snap_common, 'log'))
        fs.makepath(join(self.snap_common, 'nginx'))
        fs.makepath(join(self.snap_common, 'db'))

        storage.init_storage(APP_NAME, USER_NAME)

        templates_path = join(self.app_dir, 'config')

        variables = {
            'app': APP_NAME,
            'app_dir': self.app_dir,
            'snap_data': self.snap_data,
            'snap_common': self.snap_common,
            'device_domain_name': self.device_domain_name
        }
        gen.generate_files(templates_path, self.config_path, variables)

    def init_keys(self):
        fs.makepath(self.openvpn_config_dir)
        openvpn_keys_dir = join(self.openvpn_config_dir, 'keys')
        fs.makepath(openvpn_keys_dir)
        shutil.copy(join(self.config_path, 'openvpn/keys/index.txt'), openvpn_keys_dir)
        shutil.copy(join(self.config_path, 'openvpn/keys/serial'), openvpn_keys_dir)
        shutil.copy(join(self.config_path, 'openvpn/keys/vars'), openvpn_keys_dir)
        check_output('{0} dhparam -dsaparam -out {1}/dh2048.pem 2048'.format(
            self.openssl_bin, self.openvpn_config_dir), shell=True)
        check_output(self.generate_keys_bin, shell=True)


    def fix_permissions(self):
        fs.chownpath(self.snap_data, USER_NAME, recursive=True)
        fs.chownpath(self.snap_common, USER_NAME, recursive=True)

    def install(self):
        self.install_config()
        self.init_keys()
        self.fix_permissions()

    def post_refresh(self):
        self.install_config()
        self.fix_permissions()

    def configure(self):
        self.prepare_storage()
        install_file = join(self.snap_common, 'installed')
        if not isfile(install_file):
            fs.touchfile(install_file)
        # else:
            # upgrade

    def on_disk_change(self):
        self.prepare_storage()

    def prepare_storage(self):
        storage.init_storage(APP_NAME, USER_NAME)
