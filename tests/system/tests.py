import time
import xmlrpc.client
import subprocess

import pytest

import containers
from containers import KEA_1_6, KEA_LATEST

SUPPORTED_DISTROS = [
    ('ubuntu/18.04', 'centos/7'),
    ('centos/7', 'ubuntu/18.04')
]


def banner(txt):
    print("=" * 80)
    print(txt)


def _get_machines(server, agent, authorized=None, expected_items=None):
    # get machine that automaticaly registered in the server and authorize it
    url = '/machines'
    if authorized is not None:
        url += '?authorized=%s' % ('true' if authorized else 'false')
    for i in range(100):
        r = server.api_get(url)
        data = r.json()
        if expected_items is None:
            break
        if 'items' in data and data['items'] and len(data['items']) > 0:
            break
        time.sleep(2)
    if expected_items is not None:
        assert 'items' in data
        assert data['items'] is not None
        assert len(data['items']) == expected_items
    return data['items']


def _get_machine_and_authorize_it(server, agent):
    # get machine that automaticaly registered in the server and authorize it
    machines = _get_machines(server, agent, authorized=None, expected_items=1)
    m = machines[0]
    machine = dict(
        address=m['address'],
        agentPort=m['agentPort'],
        authorized=True)
    r = server.api_put('/machines/%d' % m['id'], json=machine, expected_status=200)
    data = r.json()
    assert agent.mgmt_ip is not None
    assert data['address'] == agent.mgmt_ip
    assert data['authorized']
    return data


def _get_machine_state(server, m_id):
    for i in range(100):
        r = server.api_get('/machines/%d/state' % m_id)
        data = r.json()
        if data['apps'] and len(data['apps'][0]['details']['daemons']) > 1:
            break
        time.sleep(2)
    return data


@pytest.mark.parametrize("agent, server", SUPPORTED_DISTROS)
def test_users_management(agent, server):
    """Check if users can be fetched and added."""
    # login
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


@pytest.mark.parametrize("distro_agent, distro_server", SUPPORTED_DISTROS)
def test_pkg_upgrade_agent_token(distro_agent, distro_server):
    """Check if Stork agent and server can be upgraded from latest release
    to localy built packages."""
    server = containers.StorkServerContainer(alias=distro_server)
    agent = containers.StorkAgentContainer(alias=distro_agent)

    # install the latest version of stork from cloudsmith
    server.setup_bg('cloudsmith')
    while server.mgmt_ip is None:
        time.sleep(0.1)
    agent.setup_bg('cloudsmith', server.mgmt_ip)
    server.setup_wait()
    agent.setup_wait()

    # install local packages
    banner('UPGRADING STORK')
    server.prepare_stork_server()
    agent.prepare_stork_agent()

    # install kea on the agent machine
    agent.install_kea()

    # login
    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    # get machine that automaticaly registered in the server and authorize it
    m = _get_machine_and_authorize_it(server, agent)

    # retrieve state of machines
    m = _get_machine_state(server, m['id'])
    assert m['apps'] is not None
    assert len(m['apps']) == 1
    assert m['apps'][0]['version'] == KEA_LATEST.split('-')[0]


@pytest.mark.parametrize("distro_agent, distro_server", SUPPORTED_DISTROS)
def test_pkg_upgrade_server_token(distro_agent, distro_server):
    """Check if Stork agent and server can be upgraded from latest release
    to localy built packages."""
    server = containers.StorkServerContainer(alias=distro_server)
    agent = containers.StorkAgentContainer(alias=distro_agent)

    # install the latest version of stork from cloudsmith
    server.setup_bg('cloudsmith')
    while server.mgmt_ip is None:
        time.sleep(0.1)
    agent.setup_bg('cloudsmith', server.mgmt_ip)
    server.setup_wait()
    agent.setup_wait()

    # login
    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    # install local packages
    banner('UPGRADING STORK SERVER')
    server.prepare_stork_server()

    # get server token from server
    for i in range(100):
        try:
            r = server.api_get('/machines-server-token')
            break
        except:
            if i == 99:
                raise
        time.sleep(1)
    data = r.json()
    server_token = data['token']

    # install kea on the agent machine
    agent.install_kea()

    # install local packages using server token based way
    banner('UPGRADING STORK AGENT')
    server_url = 'http://%s:8080' % server.mgmt_ip
    agent.run('curl -o stork-install-agent.sh %s/stork-install-agent.sh' % server_url)
    agent.run('chmod a+x stork-install-agent.sh')
    env = {'STORK_AGENT_ADDRESS': '%s:8080' % agent.mgmt_ip,
           'STORK_AGENT_SERVER_URL': server_url,
           'STORK_AGENT_SERVER_TOKEN': server_token}
    agent.run('./stork-install-agent.sh', env=env)

    # check current machines, there should be one authorized
    machines = _get_machines(server, agent, authorized=True, expected_items=1)

    # retrieve state of machines
    m = _get_machine_state(server, machines[0]['id'])
    assert m['apps'] is not None
    assert len(m['apps']) == 1
    assert m['apps'][0]['version'] == KEA_LATEST.split('-')[0]


@pytest.mark.parametrize("agent, server", [('ubuntu/18.04', 'centos/7')])
def test_add_kea_with_many_subnets(agent, server):
    """Check if Stork agent and server will handle Kea instance with huge amount of subnets."""
    # install kea on the agent machine
    agent.install_kea()

    # prepare kea config with many subnets and upload it to the agent
    banner("UPLOAD KEA CONFIG WITH MANY SUBNETS")
    subprocess.run('../../docker/gen-kea-config.py 7000 > kea-dhcp4-many-subnets.conf', shell=True, check=True)
    agent.upload('kea-dhcp4-many-subnets.conf', '/etc/kea/kea-dhcp4.conf')
    subprocess.run('rm -f kea-dhcp4-many-subnets.conf', shell=True)
    agent.run('systemctl restart isc-kea-dhcp4-server')

    # login
    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    # get machine that automaticaly registered in the server and authorize it
    m = _get_machine_and_authorize_it(server, agent)

    # check machine state
    m = _get_machine_state(server, m['id'])
    assert m['apps'] is not None
    assert len(m['apps']) == 1
    assert m['apps'][0]['version'] == KEA_LATEST.split('-')[0]
    assert len(m['apps'][0]['accessPoints']) == 1
    assert m['apps'][0]['accessPoints'][0]['address'] == '127.0.0.1'

    # get subnets (wait until the whole kea config is loaded to stork server)
    # and check if total is huge enough
    banner("GET SUBNETS")
    for i in range(30):
        r = server.api_get('/subnets?start=0&limit=10')
        data = r.json()
        if 'total' in data and data['total'] == 6912:
            break
        time.sleep(2)
    assert data['total'] == 6912


def _wait_for_event(server, text):
    last_ts = None
    event_occured = False
    for i in range(20):
        r = server.api_get('/events')
        data = r.json()
        for ev in reversed(data['items']):
            if last_ts and ev['createdAt'] < last_ts:
                # skip older events
                continue
            if text in ev['text']:
                event_occured = True
                break
        if event_occured:
            break
        last_ts = data['items'][0]['createdAt']
        time.sleep(2)
    assert event_occured, 'no event about `%s`' % text


@pytest.mark.parametrize("agent, server", [('centos/7', 'ubuntu/18.04')])
def test_change_kea_ca_access_point(agent, server):
    """Check if Stork server notices that Kea CA has changed its listening address."""
    # install kea on the agent machine
    agent.install_kea()

    # login
    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    # get machine that automaticaly registered in the server and authorize it
    m = _get_machine_and_authorize_it(server, agent)

    # check machine state
    m = _get_machine_state(server, m['id'])
    assert m['apps'] is not None
    assert len(m['apps']) == 1
    assert m['apps'][0]['version'] == '1.8.0'
    assert len(m['apps'][0]['accessPoints']) == 1
    assert m['apps'][0]['accessPoints'][0]['address'] == '127.0.0.1'

    # stop and reconfigure CA to serve from different IP address
    banner("STOP CA and reconfigure listen IP address")
    agent.run('sed -i -e s/"0.0.0.0"/"%s"/g /etc/kea/kea-ctrl-agent.conf' % agent.mgmt_ip)
    ca_svc_name = 'kea-ctrl-agent' if 'centos' in agent.name else 'isc-kea-ctrl-agent'
    agent.run('systemctl stop ' + ca_svc_name)

    # wait for unreachable event
    banner("WAIT FOR UNREACHABLE EVENT")
    _wait_for_event(server, 'is unreachable')

    # start CA
    banner("START CA")
    agent.run('systemctl start ' + ca_svc_name)

    # wait for reachable event
    banner("WAIT FOR REACHABLE EVENT")
    _wait_for_event(server, 'is reachable now')

    # check for sure if app has new access point address
    banner("CHECK IF RECONFIGURED")
    machines = _get_machines(server, agent, authorized=True, expected_items=1)
    m = machines[0]
    assert m['apps'] is not None
    assert len(m['apps']) == 1
    assert len(m['apps'][0]['accessPoints']) == 1
    assert m['apps'][0]['accessPoints'][0]['address'] == agent.mgmt_ip


def run_perfdhcp(src_cntr, dest_ip_addr):
    cmd = '/usr/sbin/perfdhcp -4 -r %d -R %d -p 10 -b mac=%s:00:00:00:00 %s' % (10, 10000, '00:00', dest_ip_addr)
    result = src_cntr.run(cmd, ignore_error=True)
    if result[0] not in [0, 3]:
        raise Exception('perfdhcp erred: %s' % str(result))


# TODO: the test is disabled for now because it does not work on GitLab CI because whole network inside LXD
# is IPv6 but it is needed to be IPv4
@pytest.mark.parametrize("agent_kea, agent_old_kea, server", [('ubuntu/20.04', 'ubuntu/18.04', 'centos/7')])
def atest_get_kea_stats(agent_kea, agent_old_kea, server):
    elems = [(agent_kea, KEA_LATEST, agent_old_kea.mgmt_ip),
             (agent_old_kea, KEA_1_6, agent_kea.mgmt_ip)]
    for idx, (a, ver, relay_addr) in enumerate(elems):
        a.install_kea(kea_version=ver)
        a.upload('kea-dhcp4.conf', '/etc/kea/kea-dhcp4.conf')
        # set proper relay address
        a.run('sed -i -e s/172\.100\.0\.200/%s/g /etc/kea/kea-dhcp4.conf' % relay_addr)
        # differentiate subnets which are used in this test
        prefix = '192.%d.2' % idx
        a.run('sed -i -e s/192.0.2/%s/g /etc/kea/kea-dhcp4.conf' % prefix)
        a.run('systemctl restart isc-kea-dhcp4-server')

    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    # add machine: kea and old_kea
    banner("ADD MACHINES")
    machines = []
    for addr in [agent_kea.mgmt_ip, agent_old_kea.mgmt_ip]:
        machine = dict(
            address=addr,
            agentPort=8080,
            agentCSR='TODO')
        r = server.api_post('/machines', json=machine, expected_status=200)  # TODO: POST should return 201
        m = r.json()
        assert m['address'] == addr
        machines.append(m)

    # wait for discovering apps
    banner("WAIT FOR DISCOVERING APPS")
    for i in range(60):
        # force refreshing machines' state
        for m in machines:
            server.api_get('/machines/%d/state' % m['id'])
        # get machines and check if there is expected data
        r = server.api_get('/machines')
        data = r.json()
        if len(data['items']) == 2 and data['items'][0]['apps'] is not None and data['items'][1]['apps'] is not None:
            break
        time.sleep(2)

    # check apps
    for m in data['items']:
        assert m['apps'] and len(m['apps']) == 1
    latest_ver = KEA_LATEST.split('-')[0]
    assert ((data['items'][0]['apps'][0]['version'] == '1.6.3' and data['items'][1]['apps'][0]['version'] == latest_ver) or
            (data['items'][0]['apps'][0]['version'] == latest_ver and data['items'][1]['apps'][0]['version'] == '1.6.3'))

    # send DHCP traffic to Kea apps
    banner("SEND DHCP TRAFFIC TO KEA APPS")

    # send DHCP traffic to old kea
    agent_kea.run('systemctl stop isc-kea-dhcp4-server')
    run_perfdhcp(agent_kea, agent_old_kea.mgmt_ip)
    agent_kea.run('systemctl start isc-kea-dhcp4-server')

    # send DHCP traffic to new kea
    agent_old_kea.run('systemctl stop isc-kea-dhcp4-server')
    run_perfdhcp(agent_old_kea, agent_kea.mgmt_ip)
    agent_old_kea.run('systemctl start isc-kea-dhcp4-server')

    # check gathered stats by Stork server
    for i in range(80):
        r = server.api_get('/overview')
        data = r.json()
        if data['dhcp4Stats'] and 'assignedAddresses' in data['dhcp4Stats'] and data['dhcp4Stats']['assignedAddresses'] > 150:
            break
        time.sleep(2)

    assert data['dhcp4Stats']
    assert 'assignedAddresses' in data['dhcp4Stats']
    assert data['dhcp4Stats']['assignedAddresses'] > 150
    assert 'subnets4' in data
    assert data['subnets4']['items']

    # there should be 2 subnets where are assigned addresses
    sn_with_addrs = 0
    for sn in data['subnets4']['items']:
        print('=== SN %s: %s' % (sn['subnet'], str(sn)))
        if sn['subnet'] not in ['192.0.2.0/24', '192.1.2.0/24']:
            continue
        for lsn in sn['localSubnets']:
            print('--- LSN %s' % str(lsn))
            if 'stats' not in lsn:
                continue
            stats = lsn['stats']
            if 'assigned-addresses' in stats and stats['assigned-addresses'] > 70:
                sn_with_addrs += 1
    assert sn_with_addrs == 2
