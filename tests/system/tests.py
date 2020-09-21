import time
import xmlrpc.client
import subprocess

import pytest

import containers


SUPPORTED_DISTROS = [
    ('ubuntu/18.04', 'centos/7'),
    ('centos/7', 'ubuntu/18.04')
]


def banner(txt):
    print("=" * 80)
    print(txt)


@pytest.mark.parametrize("agent, server", SUPPORTED_DISTROS)
def test_users_management(agent, server):
    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    # TODO: these are crashing
    # r = server.api_get('/users')
    # r = server.api_post('/users')

    r = server.api_get('/users', params=dict(start=0, limit=10))
    #assert r.json() == {"items":[{"email":"","groups":[1],"id":1,"lastname":"admin","login":"admin","name":"admin"}],"total":1}

    r = server.api_get('/groups', params=dict(start=0, limit=10))
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
    r = server.api_post('/users', json=user, expected_status=200)  # TODO: POST should return 201


@pytest.mark.parametrize("agent, server", SUPPORTED_DISTROS)
def test_machines(agent, server):
    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    machine = dict(
        address=agent.mgmt_ip,
        agentPort=8080)
    r = server.api_post('/machines', json=machine, expected_status=200)  # TODO: POST should return 201
    assert r.json()['address'] == agent.mgmt_ip

    time.sleep(10)

    r = server.api_get('/machines')
    data = r.json()
    for m in data['items']:
        assert m['apps'] is not None
        assert len(m['apps']) == 1
        assert m['apps'][0]['version'] == '1.7.3'
        # assert m['apps'][0]['version'] == '1.8.0'


@pytest.mark.parametrize("distro_agent, distro_server", SUPPORTED_DISTROS)
def test_pkg_upgrade(distro_agent, distro_server):
    server = containers.StorkServerContainer(alias=distro_server)
    agent = containers.StorkAgentContainer(alias=distro_agent)

    # install the latest version of stork from cloudsmith
    server.setup_bg('cloudsmith')
    agent.setup_bg('cloudsmith')
    server.setup_wait()
    agent.setup_wait()

    time.sleep(3)

    # install local packages
    banner('UPGRADING STORK')
    agent.prepare_stork_agent()
    server.prepare_stork_server()

    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    machine = dict(
        address=agent.mgmt_ip,
        agentPort=8080)
    r = server.api_post('/machines', json=machine, expected_status=200)  # TODO: POST should return 201
    assert r.json()['address'] == agent.mgmt_ip

    for i in range(100):
        r = server.api_get('/machines')
        data = r.json()
        if len(data['items']) == 1 and len(data['items'][0]['apps'][0]['details']['daemons']) > 1:
            break
        time.sleep(2)

    m = data['items'][0]
    for d in m['apps'][0]['details']['daemons']:
        print('daemon: %s' % str(d))
    assert m['apps'] is not None
    assert len(m['apps']) == 1
    assert m['apps'][0]['version'] == '1.7.3'


@pytest.mark.parametrize("agent, server", [('ubuntu/18.04', 'centos/7')])
def test_add_kea_with_many_subnets(agent, server):
    # prepare kea config with many subnets and upload it to the agent
    banner("UPLOAD KEA CONFIG WITH MANY SUBNETS")
    subprocess.run('../../docker/gen-kea-config.py 7000 > kea-dhcp4-many-subnets.conf', shell=True, check=True)
    agent.upload('kea-dhcp4-many-subnets.conf', '/etc/kea/kea-dhcp4.conf')
    subprocess.run('rm -f kea-dhcp4-many-subnets.conf', shell=True)
    agent.run('systemctl restart isc-kea-dhcp4-server')

    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    # add machine
    banner("ADD MACHINE")
    machine = dict(
        address=agent.mgmt_ip,
        agentPort=8080)
    r = server.api_post('/machines', json=machine, expected_status=200)  # TODO: POST should return 201
    assert r.json()['address'] == agent.mgmt_ip

    for i in range(20):
        r = server.api_get('/machines')
        data = r.json()
        if len(data['items']) == 1:
            break
        time.sleep(2)
    assert len(data['items']) == 1
    m = data['items'][0]
    assert m['apps'] is not None
    assert len(m['apps']) == 1
    assert m['apps'][0]['version'] == '1.7.3'
    assert len(m['apps'][0]['accessPoints']) == 1
    assert m['apps'][0]['accessPoints'][0]['address'] == '127.0.0.1'

    banner("GET SUBNETS")
    for i in range(30):
        r = server.api_get('/subnets?start=0&limit=10')
        data = r.json()
        print(data)
        if 'total' in data and data['total'] == 6912:
            break
        time.sleep(2)
    assert data['total'] == 6912
        # assert m['apps'][0]['version'] == '1.8.0'
