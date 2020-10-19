import time

import pytest

import containers


SUPPORTED_DISTROS = [
    ('ubuntu/18.04', 'centos/7'),
    ('centos/7', 'ubuntu/18.04')
]


def prepare_one_server_and_agent(agent_distro, server_distro):
    s = containers.StorkServerContainer(alias=server_distro)
    a = containers.StorkAgentContainer(alias=agent_distro)

    s.setup_bg()
    a.setup_bg()
    s.setup_wait()
    a.setup_wait()

    time.sleep(3)

    return s, a


@pytest.mark.parametrize("agent_distro,server_distro", SUPPORTED_DISTROS)
def test_users_management(agent_distro, server_distro):
    s, a = prepare_one_server_and_agent(agent_distro, server_distro)

    r = s.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
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


@pytest.mark.parametrize("agent_distro,server_distro", SUPPORTED_DISTROS)
def test_machines(agent_distro, server_distro):
    s, a = prepare_one_server_and_agent(agent_distro, server_distro)

    r = s.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    machine = dict(
        address=a.mgmt_ip,
        agentPort=8080)
    r = s.api_post('/machines', json=machine, expected_status=200)  # TODO: POST should return 201
    assert r.json()['address'] == a.mgmt_ip

    time.sleep(10)

    r = s.api_get('/machines')
    data = r.json()
    for m in data['items']:
        print(m)

        assert m['apps'] is not None
        assert len(m['apps']) == 1
        assert m['apps'][0]['version'] == '1.7.3'
        # assert m['apps'][0]['version'] == '1.8.0'


@pytest.mark.parametrize("agent_distro,server_distro", SUPPORTED_DISTROS)
def test_pkg_upgrade(agent_distro, server_distro):
    s = containers.StorkServerContainer(alias=server_distro)
    a = containers.StorkAgentContainer(alias=agent_distro)

    # install the latest version of stork from cloudsmith
    s.setup_bg('cloudsmith')
    a.setup_bg('cloudsmith')
    s.setup_wait()
    a.setup_wait()

    time.sleep(3)

    # install local packages
    a.prepare_stork_agent()
    s.prepare_stork_server()

    r = s.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    machine = dict(
        address=a.mgmt_ip,
        agentPort=8080)
    r = s.api_post('/machines', json=machine, expected_status=200)  # TODO: POST should return 201
    assert r.json()['address'] == a.mgmt_ip

    time.sleep(10)

    r = s.api_get('/machines')
    data = r.json()
    for m in data['items']:
        print(m)

        assert m['apps'] is not None
        assert len(m['apps']) == 1
        assert m['apps'][0]['version'] == '1.7.3'
        # assert m['apps'][0]['version'] == '1.8.0'
