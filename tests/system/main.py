#!/usr/bin/env python3

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



class Container:
    def __init__(self, name, version, port, alias=DEFAULT_SYSTEM_IMAGE):
        self.name = name
        self.version = version
        self.alias = alias
        self.style = STYLES.pop()
        print(colors.color('%s: %s' % (name, str(self.style)), **self.style))
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
            line = re.sub('\x1b\[[0-9;]*[a-zA-Z]', '', line)
            # remove control characters
            line = "".join(ch for ch in line if unicodedata.category(ch)[0] != "C")
            if not line:
                continue
            prefix = '%15s:%s' % (self.name, output)
            prefix = colors.color(prefix, **self.style) + colors.color(':', fg='white', style='bold')
            print('%s %s' % (prefix, line))

    def run(self, cmd, env=None, ignore_error=False):
        cmd2 = shlex.split(cmd)
        result = self.cntr.execute(cmd2, env)
        out = 'run: %s\n' % cmd
        out += result[1]
        self._trace_logs(out, 'out')
        self._trace_logs(result[2], 'err')
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
        self.cntr.stop(wait=True)
        image = self.cntr.publish(True, True)
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


class StorkServerContainer(Container):
    def __init__(self, name, port=8932, alias=DEFAULT_SYSTEM_IMAGE):
        super().__init__(name, 1, port, alias)
        self.port = port
        self.session = requests.Session()

    def prepare_system(self):
        self.run('apt-get update')
        self.run('apt-get install -y --no-install-recommends postgresql-client postgresql-all', {'DEBIAN_FRONTEND': 'noninteractive', 'TERM': 'linux'})
        self.run('systemctl enable postgresql.service')
        self.run('systemctl start postgresql.service')
        self.run('systemctl status postgresql.service')

    def prepare_stork_db(self):
        self.run('systemctl stop isc-stork-server', ignore_error=True)
        cmd = "bash -c \"cd /tmp && cat <<EOF | sudo -u postgres psql postgres\n"
        cmd += "DROP DATABASE IF EXISTS stork;\n"
        cmd += "DROP USER IF EXISTS stork;\n"
        cmd += "CREATE USER stork WITH PASSWORD 'stork';\n"
        cmd += "CREATE DATABASE stork;\n"
        cmd += "GRANT ALL PRIVILEGES ON DATABASE stork TO stork;\n"
        cmd += "\c stork;\n"
        cmd += "create extension pgcrypto;\n"
        cmd += "EOF\n\""
        self.run(cmd)

    def prepare_stork_server(self, pkg_path):
        self.upload(pkg_path, '/root/isc-stork-server.deb')

        self.run('apt install /root/isc-stork-server.deb', {'DEBIAN_FRONTEND': 'noninteractive', 'TERM': 'linux'})
        self.run("perl -pi -e 's/.*STORK_DATABASE_PASSWORD.*/STORK_DATABASE_PASSWORD=stork/g' /etc/stork/server.env")
        self.run("perl -pi -e 's/.*STORK_REST_HOST.*/STORK_REST_HOST=localhost/g' /etc/stork/server.env")
        self.run('dpkg -l "isc-stork*"')
        self.run('systemctl enable isc-stork-server')
        self.run('systemctl start isc-stork-server')
        self.run('systemctl status isc-stork-server')
        self.run('bash -c "ps axu|grep isc"')

    def _setup(self, pkg_path):
        reused = self.start()
        time.sleep(5)
        if not reused:
            self.prepare_system()
            self.prepare_stork_db()
            self.dump_image()
        self.prepare_stork_server(pkg_path)

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
    def __init__(self, name, port=8933, alias=DEFAULT_SYSTEM_IMAGE):
        super().__init__(name, 1, port, alias)
        if 'centos' in self.alias:
            self.pkg_format = 'rpm'
        else:
            self.pkg_format = 'deb'

    def prepare_system(self):
        if self.pkg_format == 'deb':
            self.run('apt-get update')
            self.run('apt-get install --no-install-recommends -y curl', {'DEBIAN_FRONTEND': 'noninteractive', 'TERM': 'linux'})
        else:
            self.run('yum install -y curl')

    def install_kea(self):
        if self.pkg_format == 'deb':
            self.run("curl -1sLf -o cloudsmith.sh 'https://dl.cloudsmith.io/public/isc/kea-1-7/cfg/setup/bash.deb.sh'")
        else:
            self.run("curl -1sLf -o cloudsmith.sh 'https://dl.cloudsmith.io/public/isc/kea-1-7/cfg/setup/bash.rpm.sh'")
        self.run("chmod a+x cloudsmith.sh")
        self.run("./cloudsmith.sh")
        kea_version = '1.7.3-isc0009420191217090201'
        if self.pkg_format == 'deb':
            self.run("apt-get update")
            cmd = "apt-get install -y --no-install-recommends"
            cmd += " isc-kea-dhcp4-server={kea_version} isc-kea-ctrl-agent={kea_version} isc-kea-common={kea_version}"
        else:
            self.run('yum install -y epel-release perl')
            cmd = "yum install -y"
            cmd += " isc-kea-{kea_version}.el7 isc-kea-hooks-{kea_version}.el7 isc-kea-libs-{kea_version}.el7"

        cmd = cmd.format(kea_version=kea_version)
        self.run(cmd)
        self.run("mkdir -p /var/run/kea/")
        self.run("perl -pi -e 's/127\.0\.0\.1/0\.0\.0\.0/g' /etc/kea/kea-ctrl-agent.conf")
        if self.pkg_format == 'deb':
            self.run('systemctl enable isc-kea-ctrl-agent')
            self.run('systemctl start isc-kea-ctrl-agent')
            self.run('systemctl status isc-kea-ctrl-agent')
        else:
            for cmd in ['enable', 'start', 'status']:
                self.run('systemctl %s kea-dhcp4' % cmd)
                self.run('systemctl %s kea-ctrl-agent' % cmd)

    def prepare_stork_agent(self, pkg_path):
        if self.pkg_format == 'deb':
            self.upload(pkg_path, '/root/isc-stork-agent.deb')
            self.run('apt install /root/isc-stork-agent.deb', {'DEBIAN_FRONTEND': 'noninteractive', 'TERM': 'linux'})
            self.run('dpkg -l "isc-stork*"')
        else:
            self.upload(pkg_path, '/root/isc-stork-agent.rpm')
            self.run('yum install -y /root/isc-stork-agent.rpm')
            self.run('rpm -qa "isc-stork*"')
        self.run('systemctl enable isc-stork-agent')
        self.run('systemctl start isc-stork-agent')
        self.run('systemctl status isc-stork-agent')
        self.run('bash -c "ps axu|grep isc"')

    def _setup(self, pkg_path):
        reused = self.start()
        time.sleep(5)
        if not reused:
            self.prepare_system()
            self.install_kea()
            self.dump_image()
        if self.pkg_format == 'deb':
            # workaround for not starting autonomously CA service
            self.run('systemctl start isc-kea-ctrl-agent')
        self.prepare_stork_agent(pkg_path)


def main(srv_pkg_path_deb, agn_pkg_path_deb, srv_pkg_path_rpm, agn_pkg_path_rpm):
    s = StorkServerContainer('stork-server')
    a = StorkAgentContainer('stork-agent')
    a_c7 = StorkAgentContainer('stork-agent-c7', port=8934, alias='centos/7')

    s.setup_bg(srv_pkg_path_deb)
    a.setup_bg(agn_pkg_path_deb)
    a_c7.setup_bg(agn_pkg_path_rpm)
    s.setup_wait()
    a.setup_wait()
    a_c7.setup_wait()

    time.sleep(3)


    r = s.api_post('/sessions', params=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    # TODO: these are crashing
    # r = s.api_get('/users')
    # r = s.api_post('/users')

    r = s.api_get('/users', params=dict(start=0, limit=10))
    #assert r.json() == {"items":[{"email":"","groups":[1],"id":1,"lastname":"admin","login":"admin","name":"admin"}],"total":1}

    r = s.api_get('/groups', params=dict(start=0, limit=10))
    groups = r.json()
    assert groups['total'] == 2
    assert len(groups['items']) == 2
    assert groups['items'][0]['name'] in ['super-admin', 'admin']
    assert groups['items'][1]['name'] in ['super-admin', 'admin']

    user = dict(
        user=dict(id=0,
                  login='user',
                  email='a@example.org',
                  name='John',
                  lastname='Smith',
                  groups=[]),
        password='password')
    r = s.api_post('/users', json=user, expected_status=200)  # TODO: POST should return 201

    for m in [a, a_c7]:
        machine = dict(
            address=m.mgmt_ip,
            agentPort=8080)
        r = s.api_post('/machines', json=machine, expected_status=200)  # TODO: POST should return 201
        assert r.json()['address'] == m.mgmt_ip

    #time.sleep(10)

    r = s.api_get('/machines')
    data = r.json()
    for m in data['items']:
        print(m)

        assert m['apps'] is not None
        assert len(m['apps']) == 1
        assert m['apps'][0]['version'] == '1.7.3'



if __name__ == '__main__':
    main(sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4])
