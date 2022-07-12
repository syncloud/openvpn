import logging
import os
import shutil
from os.path import join, isfile
from subprocess import check_output

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
        self.prefix_delegation_bin = join(self.app_dir, 'bin/prefix_delegation.sh')
        self.prefix_delegation_link = join(self.app_dir, '/etc/dhcp/dhclient-exit-hooks.d/openvpn')
        self.device_domain_name = urls.get_device_domain_name()
        self.pki_dir = join(self.snap_data, 'pki')
        self.dh_file = join(self.openvpn_config_dir, 'dh2048.pem')
        self.pki_private_dir = join(self.pki_dir, 'private')
        self.ca_file = join(self.pki_dir, 'ca.crt')
        self.server_cert_file = join(self.pki_dir, 'issued', 'server.crt')
        self.server_key_file = join(self.pki_dir, 'private', 'server.key')
        self.server_conf_file = join(self.openvpn_config_dir, 'server.conf')

    def install_config(self):

        home_folder = join('/home', USER_NAME)
        linux.useradd(USER_NAME, home_folder=home_folder)

        fs.makepath(join(self.snap_common, 'log'))
        fs.makepath(join(self.snap_common, 'nginx'))
        fs.makepath(join(self.snap_common, 'db'))
        fs.makepath(self.openvpn_config_dir)
        fs.makepath(self.pki_dir)
        fs.makepath(self.pki_private_dir)
        fs.makepath(join(self.pki_dir, 'reqs'))

        if os.path.lexists(self.prefix_delegation_link):
            os.remove(self.prefix_delegation_link)
        os.symlink(self.prefix_delegation_bin, self.prefix_delegation_link)

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

        if not os.path.exists(join(self.pki_dir, 'index.txt')):
            shutil.copy(join(self.config_path, 'pki/index.txt'), self.pki_dir)
        if not os.path.exists(join(self.pki_dir, 'serial')):
            shutil.copy(join(self.config_path, 'pki/serial'), self.pki_dir)
        if not os.path.exists(self.dh_file):
            check_output('{0} dhparam -dsaparam -out {1} 2048'.format(self.openssl_bin, self.dh_file), shell=True)
        
    def fix_permissions(self):
        fs.chownpath(self.snap_data, USER_NAME, recursive=True)
        fs.chownpath(self.snap_common, USER_NAME, recursive=True)

    def install(self):
        self.install_config()
        check_output(self.generate_keys_bin, shell=True)
        self.fix_permissions()

    def post_refresh(self):
        self.install_config()

        if not os.path.exists(self.ca_file):
            check_output('sed -i "s#ca .*#ca {0}#g" {1}'.format(self.ca_file, self.server_conf_file), shell=True)
            check_output('sed -i "s#cert .*#cert {0}#g" {1}'.format(self.server_cert_file, self.server_conf_file), shell=True)
            check_output('sed -i "s#key .*#key {0}#g" {1}'.format(self.server_key_file, self.server_conf_file), shell=True)
            check_output(self.generate_keys_bin, shell=True)

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
        self.log.info("prepare_storage")
        storage.init_storage(APP_NAME, USER_NAME)
