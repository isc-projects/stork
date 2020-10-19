#!/usr/bin/env python3

import os
import re
import sys
import time
import shlex
import random
import argparse
import threading
import unicodedata

from pylxd import Client
import requests
import colors

DEFAULT_SYSTEM_IMAGE = 'ubuntu/18.04'

STYLES = [dict(fg='red', style=''),
          dict(fg='green', style=''),
          dict(fg='yellow', style=''),
          dict(fg='blue', style=''),
          dict(fg='magenta', style=''),
          dict(fg='cyan', style=''),
          dict(fg='red', style='bold'),
          dict(fg='green', style='bold'),
          dict(fg='yellow', style='bold'),
          dict(fg='blue', style='bold'),
          dict(fg='magenta', style='bold'),
          dict(fg='cyan', style='bold')]
random.shuffle(STYLES)


DEFAULT_STORK_DEB_VERSION = None
DEFAULT_STORK_RPM_VERSION = None


class Container:
    def __init__(self, name, version, port, alias=DEFAULT_SYSTEM_IMAGE):
        self.name = name
        self.version = version
        self.alias = alias

        if 'centos' in self.alias:
            self.pkg_format = 'rpm'
        else:
            self.pkg_format = 'deb'

        # prepare styling for traces
        self.style = STYLES.pop()
        print(colors.color('%s: %s' % (name, str(self.style)), **self.style))

        # open separate connection to LXD
        self.lxd = Client()

        self.config = {
            'name': name,
            'source': {
                'type': 'image',
                'mode': 'pull',
#                'server': 'https://cloud-images.ubuntu.com/daily',
                'server': 'https://us.images.linuxcontainers.org/',
                'protocol': 'simplestreams',
                'alias': alias
            },
            'devices': {
                'myproxy': {
                    'type': 'proxy',
                    'listen': 'tcp:0.0.0.0:%d' % port,
                    'connect': 'tcp:127.0.0.1:8080'
                }
            }
        }
        self.cntr = None
        self.thread = None
        self.bg_exc = None

    def start(self):
        try:
            reused_img = self.lxd.images.get_by_alias(self.name)
            if int(reused_img.properties['version']) < self.version:
                reused_img.delete()
                reused_img = None
            else:
                self.config['source'] = {
                    'type': 'image',
                    'alias': self.name
                }
                print('reused image for %s: %s' % (self.name, reused_img.fingerprint))
        except:
            reused_img = None

        self.cntr = None
        if self.lxd.containers.exists(self.config['name']):
            c = self.lxd.containers.get(self.config['name'])
            if c.status == 'Running':
                c.stop(wait=True)
            c.delete(wait=True)

        if not self.cntr:
            self.cntr = self.lxd.containers.create(self.config, wait=True)

        if self.cntr.status != 'Running':
            self.cntr.start(wait=True)

        # wait for network address
        time.sleep(5)

        self.mgmt_ip = self.cntr.state().network['eth0']['addresses'][0]['address']

        return reused_img

    def upload(self, local_path, remote_path):
        with open(local_path, 'rb') as f:
            data = f.read()
        self.cntr.files.put(remote_path, data)

    def _trace_logs(self, log, output):
        for line in log.splitlines():
            line = line.rstrip()
            # remove ANSI escape sequences
            line = re.sub(r'\x1b\[[0-9;]*[a-zA-Z]', '', line)
            # remove control characters
            line = "".join(ch for ch in line if unicodedata.category(ch)[0] != "C")
            if not line:
                continue
            prefix = '%15s:%s' % (self.name, output)
            prefix = colors.color(prefix, **self.style) + colors.color(':', fg='white', style='bold')
            # ignore encoding errors
            line = line.encode('utf-8', errors='ignore').decode('ascii', errors='ignore')

            print('%s %s' % (prefix, line))

    def run(self, cmd, env=None, ignore_error=False, attempts=1, sleep_time_after_attempt=None):
        cmd2 = shlex.split(cmd)

        if env is None:
            env = {}
        env['LANG'] = "en_US.UTF-8"
        env['LANGUAGE'] = "en_US:UTF-8"
        env['LC_ALL'] = "en_US.UTF-8"

        for attempt in range(attempts):
            result = self.cntr.execute(cmd2, env)
            out = 'run: %s\n' % cmd
            out += result[1]
            self._trace_logs(out, 'out')
            self._trace_logs(result[2], 'err')
            if result[0] == 0:
                break
            elif attempt < attempts - 1:
                print('command failed, retry, attempt %d/%d' % (attempt, attempts))
                if sleep_time_after_attempt:
                    time.sleep(sleep_time_after_attempt)

        if result[0] != 0 and not ignore_error:
            raise Exception('problem with cmd: %s' % cmd)

    def setup_bg(self, *args):
        if self.thread is not None:
            raise Exception('there is already running bg thread for %s' % self.name)
        self.thread = threading.Thread(target=self.setup, args=args)
        self.thread.start()

    def setup_wait(self):
        self.thread.join()
        if self.bg_exc:
            print("problem with container %s" % self.name)
            e = self.bg_exc
            self.bg_exc = None
            raise e

    def setup(self, *args):
        try:
            self._setup(*args)
        except Exception as e:
            self.bg_exc = e

    def dump_image(self):
        print('dumping %s ...' % self.name)
        self.cntr.stop(wait=True)

        # there is an issue with publishing container: https://github.com/lxc/pylxd/issues/404
        # the workaround is to set type to 'container'
        old_type = self.cntr.type
        self.cntr.type = 'container'

        image = self.cntr.publish(True, True)

        # restore container type
        self.cntr.type = old_type

        os_name, release = self.alias.split('/')
        image.properties = {
            'version': str(self.version),
            'os': os_name,
            'release': release,
            'description': '%s %s, version: %d' % (os_name, release, self.version)
        }
        image.save()
        self.cntr.start(wait=True)
        try:
            old_img = self.lxd.images.get_by_alias(self.name)
            old_img.delete(wait=True)
        except:
            pass
        image.add_alias(self.name, '')
        print('dumped %s, fingerprint: %s, alias: %s' % (self.name, image.fingerprint, self.name))
        time.sleep(5)

    def setup_cloudsmith_repo(self, name):
        if self.pkg_format == 'deb':
            self.run("curl -1sLf -o cloudsmith.sh 'https://dl.cloudsmith.io/public/isc/%s/cfg/setup/bash.deb.sh'" % name)
        else:
            self.run("curl -1sLf -o cloudsmith.sh 'https://dl.cloudsmith.io/public/isc/%s/cfg/setup/bash.rpm.sh'" % name)
        self.run("chmod a+x cloudsmith.sh")
        self.run("./cloudsmith.sh")

        if self.pkg_format == 'deb':
            self.run("apt-get update")

    def install_pkgs(self, names):
        if self.pkg_format == 'deb':
            cmd = "apt-get install -y --no-install-recommends"
            env = {'DEBIAN_FRONTEND': 'noninteractive', 'TERM': 'linux'}
        else:
            cmd = "yum install -y"
            env = None
        cmd += " " + names
        self.run(cmd, env=env, attempts=3, sleep_time_after_attempt=3)

    def uninstall_pkgs(self, names):
        if self.pkg_format == 'deb':
            cmd = "apt-get purge -y"
            env = {'DEBIAN_FRONTEND': 'noninteractive', 'TERM': 'linux'}
        else:
            cmd = "yum erase -y"
            env = None
        cmd += " " + names
        self.run(cmd, env=env, attempts=3, sleep_time_after_attempt=3)

    def set_locale(self):
        if self.pkg_format == 'deb':
            self.run('locale-gen --purge en_US.UTF-8')
            cmd = "echo -e 'LANG=\"en_US.UTF-8\"\nLANGUAGE=\"en_US:UTF-8\"\n' > /etc/default/locale"
            self.run('bash -c "%s"' % cmd)
            self.run('dpkg-reconfigure -f noninteractive locales')
        else:
            #self.run('yum install -y langpacks-en')
            cmd = "echo -e 'LANG=\"en_US.UTF-8\"\nLANGUAGE=\"en_US:UTF-8\"\nLC_ALL=\"en_US.UTF-8\"\n' > /etc/profile.d/locale.sh"
            self.run('bash -c "%s"' % cmd)
            self.run('localectl set-locale LANG=en_US.UTF-8 LANGUAGE=en_US.UTF-8')


class StorkServerContainer(Container):
    def __init__(self, port=None, alias=DEFAULT_SYSTEM_IMAGE):
        if port is None:
            port = random.randrange(6000, 50000)
        name = 'stork-server-%s' % alias.replace('/', '-').replace('.', '-')
        super().__init__(name, 1, port, alias)
        self.port = port
        self.session = requests.Session()

    def prepare_system(self):
        if self.pkg_format == 'deb':
            self.run('apt-get update')
            self.set_locale()
            self.install_pkgs("postgresql-client postgresql-all curl")

            self.run('systemctl enable postgresql.service')
            self.run('systemctl start postgresql.service')
            self.run('systemctl status postgresql.service')
        else:
            self.set_locale()
            self.install_pkgs('sudo perl curl')
            #self.run('yum install -y postgresql-server postgresql-contrib sudo perl', attempts=3, sleep_time_after_attempt=3)
            #self.run('postgresql-setup initdb')
            self.run('rpm -Uvh https://yum.postgresql.org/11/redhat/rhel-7-x86_64/pgdg-redhat-repo-latest.noarch.rpm')
            self.install_pkgs('postgresql11-server postgresql11 postgresql11-contrib')
            self.run('/usr/pgsql-11/bin/postgresql-11-setup initdb')
            self.run("perl -pi -e 's/(host.*)ident/\\1md5/g'  /var/lib/pgsql/11/data/pg_hba.conf")

            self.run('systemctl enable postgresql-11.service')
            self.run('systemctl start postgresql-11.service')
            self.run('systemctl status postgresql-11.service')

    def prepare_stork_db(self):
        self.run('systemctl stop isc-stork-server', ignore_error=True)

        cmd = "bash -c \"cd /tmp && cat <<EOF | sudo -u postgres psql postgres\n"
        cmd += "DROP DATABASE IF EXISTS stork;\n"
        cmd += "DROP USER IF EXISTS stork;\n"
        cmd += "CREATE USER stork WITH PASSWORD 'stork';\n"
        cmd += "CREATE DATABASE stork;\n"
        cmd += "GRANT ALL PRIVILEGES ON DATABASE stork TO stork;\n"
        cmd += "\\c stork;\n"
        cmd += "create extension pgcrypto;\n"
        cmd += "EOF\n\""
        self.run(cmd)

    def install_stork_from_local_file(self, pkg_ver):
        if pkg_ver is None:
            if self.pkg_format == 'rpm':
                pkg_ver = DEFAULT_STORK_RPM_VERSION
            else:
                pkg_ver = DEFAULT_STORK_DEB_VERSION

        if self.pkg_format == 'rpm':
            pkg_name = 'isc-stork-server-%s-1.x86_64.rpm' % pkg_ver
        else:
            pkg_name = 'isc-stork-server_%s_amd64.deb' % pkg_ver
        pkg_path = os.path.abspath(os.path.join('../..', pkg_name))

        self.upload(pkg_path, '/root/isc-stork-server.%s' % self.pkg_format)

        if self.pkg_format == 'deb':
            self.run('apt install -y --allow-downgrades /root/isc-stork-server.deb', {'DEBIAN_FRONTEND': 'noninteractive', 'TERM': 'linux'})
        else:
            self.run('yum install -y /root/isc-stork-server.rpm')


    def prepare_stork_server(self, pkg_ver=None):
        if pkg_ver == 'cloudsmith':
            self.setup_cloudsmith_repo('stork')
            pkgs = ''
            if self.pkg_format == 'rpm':
                pkgs = 'epel-release perl'
            pkgs += ' isc-stork-server'
            self.install_pkgs(pkgs)
        else:
            self.install_stork_from_local_file(pkg_ver)

        if self.pkg_format == 'deb':
            self.run('dpkg -l "isc-stork*"')
        else:
            self.run('rpm -qa "isc-stork*"')

        self.run("perl -pi -e 's/.*STORK_DATABASE_PASSWORD.*/STORK_DATABASE_PASSWORD=stork/g' /etc/stork/server.env")
        self.run("perl -pi -e 's/.*STORK_REST_HOST.*/STORK_REST_HOST=localhost/g' /etc/stork/server.env")

        self.run('systemctl enable isc-stork-server')
        self.run('systemctl start isc-stork-server')
        #self.run('systemctl status isc-stork-server')
        self.run('bash -c "ps axu|grep isc"')

    def _setup(self, pkg_ver=None):
        reused = self.start()
        time.sleep(5)
        if not reused:
            self.prepare_system()
            self.prepare_stork_db()
            self.dump_image()
        self.prepare_stork_server(pkg_ver)

    def api_get(self, endpoint, params=None, expected_status=200):
        url = 'http://localhost:%d/api' % self.port
        url += endpoint
        r = self.session.get(url, params=params)
        print('r.status_code', r.status_code)
        print('r.text', r.text)
        assert r.status_code == expected_status
        return r

    def api_post(self, endpoint, params=None, json=None, expected_status=201):
        url = 'http://localhost:%d/api' % self.port
        url += endpoint
        r = self.session.post(url, params=params, json=json)
        print('r.status_code', r.status_code)
        print('r.text', r.text)
        assert r.status_code == expected_status
        return r


class StorkAgentContainer(Container):
    def __init__(self, port=None, alias=DEFAULT_SYSTEM_IMAGE):
        if port is None:
            port = random.randrange(6000, 50000)
        name = 'stork-agent-%s' % alias.replace('/', '-').replace('.', '-')
        super().__init__(name, 1, port, alias)

    def prepare_system(self):
        if self.pkg_format == 'deb':
            self.run('apt-get update')
        self.set_locale()
        self.install_pkgs('curl')

    def install_kea(self, service_name='default'):
        self.setup_cloudsmith_repo('kea-1-7')
        kea_version = '1.7.3-isc0009420191217090201'
        #self.setup_cloudsmith_repo('kea-1-8')
        #kea_version = '1.8.0-isc0000420200825110759'
        if self.pkg_format == 'deb':
            self.run("apt-get update")
            if service_name == 'default':
                pkgs = " isc-kea-dhcp4-server={kea_version} isc-kea-ctrl-agent={kea_version} isc-kea-common={kea_version}"
            elif 'dhcp6' in service_name:
                pkgs = " isc-kea-dhcp6-server={kea_version}"
            elif 'ddns' in service_name:
                pkgs = " isc-kea-dhcp-ddns-server={kea_version}"
            else:
                assert False, "incorrect kea service name: %s" % service_name
        else:
            self.install_pkgs('epel-release')
            pkgs = 'perl'
            pkgs += " isc-kea-{kea_version}.el7 isc-kea-hooks-{kea_version}.el7 isc-kea-libs-{kea_version}.el7"

        pkgs = pkgs.format(kea_version=kea_version)
        self.install_pkgs(pkgs)

        self.run("mkdir -p /var/run/kea/")
        self.run("perl -pi -e 's/127\\.0\\.0\\.1/0\\.0\\.0\\.0/g' /etc/kea/kea-ctrl-agent.conf")
        if self.pkg_format == 'deb':
            self.run('systemctl enable isc-kea-ctrl-agent')
            self.run('systemctl start isc-kea-ctrl-agent')
            self.run('systemctl status isc-kea-ctrl-agent')
        else:
            for cmd in ['enable', 'start', 'status']:
                self.run('systemctl %s kea-dhcp4' % cmd)
                self.run('systemctl %s kea-ctrl-agent' % cmd)

    def install_bind(self, conf_file=None):
        # this should install specific version of bind but let's keep it
        # that way for now
        if self.pkg_format == 'deb':
            self.run('apt-get install -y software-properties-common')
            self.run('add-apt-repository ppa:isc/bind -y')
            self.run('apt-get install -y bind9')
        else:
            self.run('yum -y install bind bind-utils')

        if conf_file is not None:
            self.upload(conf_file, '/etc/bind/named.conf')

        self.run('systemctl enable named')
        self.run('systemctl start named')
        self.run('systemctl status named')

    def uninstall_kea(self, pkgs_name='all'):
        if self.pkg_format == 'deb':
            if pkgs_name == 'all':
                pkgs = "isc-kea-dhcp-ddns-server isc-kea-dhcp6-server isc-kea-dhcp4-server isc-kea-ctrl-agent isc-kea-common"
            else:
                pkgs = pkgs_name
        else:
            pkgs = " isc-kea isc-kea-hooks isc-kea-libs"
        self.uninstall_pkgs(pkgs)

    def start_kea(self, process):
        if self.pkg_format == 'deb':
            process = 'isc-%s-server' % process
        for cmd in ['start', 'status']:
            self.run('systemctl %s %s' % (cmd, process))

    def stop_kea(self, process):
        if self.pkg_format == 'deb':
            process = 'isc-%s-server' % process
        self.run('systemctl stop %s' % process)
        time.sleep(1)

    def install_stork_from_local_file(self, pkg_ver=None):
        if pkg_ver is None:
            if self.pkg_format == 'rpm':
                pkg_ver = DEFAULT_STORK_RPM_VERSION
            else:
                pkg_ver = DEFAULT_STORK_DEB_VERSION

        if self.pkg_format == 'rpm':
            pkg_name = 'isc-stork-agent-%s-1.x86_64.rpm' % pkg_ver
        else:
            pkg_name = 'isc-stork-agent_%s_amd64.deb' % pkg_ver
        pkg_path = os.path.abspath(os.path.join('../..', pkg_name))

        self.upload(pkg_path, '/root/isc-stork-agent.%s' % self.pkg_format)

        if self.pkg_format == 'deb':
            self.run('apt install -y --allow-downgrades /root/isc-stork-agent.deb', {'DEBIAN_FRONTEND': 'noninteractive', 'TERM': 'linux'})
        else:
            self.run('yum install -y /root/isc-stork-agent.rpm')

    def prepare_stork_agent(self, pkg_ver=None):
        if pkg_ver == 'cloudsmith':
            self.setup_cloudsmith_repo('stork')
            self.install_pkgs('isc-stork-agent')
        else:
            self.install_stork_from_local_file(pkg_ver)

        if self.pkg_format == 'deb':
            self.run('dpkg -l "isc-stork*"')
        else:
            self.run('rpm -qa "isc-stork*"')

        self.run('systemctl enable isc-stork-agent')
        self.run('systemctl start isc-stork-agent')
        self.run('systemctl status isc-stork-agent')
        self.run('bash -c "ps axu|grep isc"')

    def _setup(self, pkg_ver=None):
        reused = self.start()
        time.sleep(5)
        if not reused:
            self.prepare_system()
            self.install_kea()
            self.dump_image()
        if self.pkg_format == 'deb':
            # workaround for not starting autonomously CA service
            self.run('systemctl start isc-kea-ctrl-agent')
        self.prepare_stork_agent(pkg_ver)
