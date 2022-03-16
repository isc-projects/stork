import os
import time
import xmlrpc.client
import subprocess

import pytest

import containers
from containers import KEA_1_6, KEA_LATEST, KeaTLSSupport

SUPPORTED_DISTROS = [
    ('ubuntu/18.04', 'centos/8'),
    ('centos/8', 'ubuntu/18.04')
]


def banner(txt):
    print("=" * 80)
    print(txt)


def _get_machines(server, authorized=None, expected_items=None, machine_filter=None):
    # get machine from the server
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

    assert 'items' in data
    machines = data['items']
    assert machines is not None

    if machine_filter is not None:
        machines = [m for m in machines if machine_filter(m)]

    if expected_items is not None:
        assert len(machines) == expected_items
    return machines


def _get_machines_and_authorize_them(server, expected_items=1, machine_filter=None):
    # get machine that automatically registered in the server and authorize it
    machines = _get_machines(server, authorized=None, expected_items=expected_items, machine_filter=machine_filter)
    machines2 = []

    for m in machines:
        machine = dict(
            address=m['address'],
            agentPort=m['agentPort'],
            authorized=True)
        r = server.api_put('/machines/%d' % m['id'], json=machine, expected_status=200)
        data = r.json()
        assert data['authorized']
        machines2.append(data)
    return machines2


def _get_machine_state(server, m_id):
    '''We have a hard to resolve race problem with update machine state in the Stork.
    The problem occurs when the Stork tries refreshing an application state from multiple goroutines at the same time.

    Refresh may be triggered by:

    - On Stork start
    - Periodically
    - On user request

    Some refresh procedures may be called in the same time.
    Refresh state looks like this:

        1. Get state from an agent
        2. Get state from a database
        3. Calculate diff
        4. Provide changes in the application
        5. Provide changes in the subnets and others

    Points 2, 4, and 5 are done in a separate transaction.
    It may happen that after fetching state from the database (2.) and calculate diffs (3.) in one goroutine,
    another goroutine modified the application. It causes that the calculated diffs are incorrect.
    The exception is thrown from point 4. where the unique index constraints are checked.
    If exception is thrown then it means that the insert conflict may occur.
    Another goroutine could correctly insert the state.
    '''
    error_attempts = 0
    for i in range(100):
        try:
            r = server.api_get('/machines/%d/state' % m_id)
        except Exception:
            if error_attempts < 3:
                error_attempts += 1
                time.sleep(5)
                continue

        data = r.json()
        if data['apps'] and data['apps'][0]['details']:
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

    # get machine that automatically registered in the server and authorize it
    m = _get_machines_and_authorize_them(server)[0]
    assert m['address'] == agent.mgmt_ip

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
    env = {'STORK_AGENT_HOST': '%s:8080' % agent.mgmt_ip,
           'STORK_AGENT_SERVER_URL': server_url,
           'STORK_AGENT_SERVER_TOKEN': server_token}
    agent.run('./stork-install-agent.sh', env=env)

    # check current machines, there should be one authorized
    machines = _get_machines(server, authorized=True, expected_items=1)

    # retrieve state of machines
    m = _get_machine_state(server, machines[0]['id'])
    assert m['apps'] is not None
    assert len(m['apps']) == 1
    assert m['apps'][0]['version'] == KEA_LATEST.split('-')[0]


@pytest.mark.parametrize("agent, server", [('ubuntu/18.04', 'centos/8')])
def test_add_kea_with_many_subnets(agent, server):
    """Check if Stork agent and server will handle Kea instance with huge amount of subnets."""
    # install kea on the agent machine
    agent.install_kea()

    # prepare kea config with many subnets and upload it to the agent
    banner("UPLOAD KEA CONFIG WITH MANY SUBNETS")
    subprocess.run('../../docker/tools/gen-kea-config.py 7000 > kea-dhcp4-many-subnets.conf', shell=True, check=True)
    agent.upload('kea-dhcp4-many-subnets.conf', '/etc/kea/kea-dhcp4.conf')
    subprocess.run('rm -f kea-dhcp4-many-subnets.conf', shell=True)
    agent.run('systemctl restart isc-kea-dhcp4-server')

    # login
    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    # get machine that automatically registered in the server and authorize it
    m = _get_machines_and_authorize_them(server)[0]
    assert m['address'] == agent.mgmt_ip

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
        # Fetch new state
        m = _get_machine_state(server, m['id'])
    assert data['total'] == 6912

    # Fetch raw Kea daemon configuration
    # TODO temporarily disable this test because it crashes piplines
    # in CI because of the too long output.
    # banner("FETCH KEA DAEMON CONFIG")
    # daemons = m['apps'][0]['details']['daemons']
    # daemons = [d for d in daemons if d['name'] == 'dhcp4']
    # assert len(daemons) == 1
    # daemon_id = daemons[0]['id']

    # r = server.api_get('/daemons/%d/config' % (daemon_id,))
    # data = r.json()

    # assert 'Dhcp4' in data
    # assert 'subnet4' in data['Dhcp4']
    # subnets = data['Dhcp4']['subnet4']
    # assert len(subnets) == 6912


def _wait_for_event(server, text, expected=True, attempts=20, details=''):
    last_ts = None
    event_occured = False
    for i in range(attempts):
        r = server.api_get('/events')
        data = r.json()
        for ev in reversed(data['items']):
            if last_ts and ev['createdAt'] < last_ts:
                # skip older events
                continue
            if text in ev['text'] and details in ev.get('details', ''):
                event_occured = True
                break
        if event_occured:
            break
        last_ts = data['items'][0]['createdAt']
        time.sleep(2)

    assert event_occured == expected, ('no' if expected else 'found') + ' event about `%s`' % text

def _search_leases(server, text=None, host_id=None):
    assert text != None or host_id != None
    leases_found = False
    data = None
    for i in range(10):
        r = None
        if text != None:
            r = server.api_get('/leases?text=%s' % text)
        elif host_id != None:
            r = server.api_get('/leases?hostId=%d' % host_id)
        data = r.json()

        if 'items' in data and data['items'] and len(data['items']) > 0:
            leases_found = True
            break
        time.sleep(2)
    assert leases_found, 'failed to find any leases by search text `%s`' % text

    if 'conflicts' in data:
        return data['items'], data['conflicts']

    return data['items'], None

@pytest.mark.parametrize("agent, server", [('centos/8', 'ubuntu/18.04')])
def test_change_kea_ca_access_point(agent, server):
    """Check if Stork server notices that Kea CA has changed its listening address."""
    # install kea on the agent machine
    agent.install_kea()

    # login
    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    # get machine that automatically registered in the server and authorize it
    m = _get_machines_and_authorize_them(server)[0]
    assert m['address'] == agent.mgmt_ip

    # check machine state
    m = _get_machine_state(server, m['id'])
    assert m['apps'] is not None
    assert len(m['apps']) == 1
    assert m['apps'][0]['version'] == KEA_LATEST.split('-')[0]
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
    machines = _get_machines(server, authorized=True, expected_items=1)
    m = machines[0]
    assert m['apps'] is not None
    assert len(m['apps']) == 1
    assert len(m['apps'][0]['accessPoints']) == 1
    assert m['apps'][0]['accessPoints'][0]['address'] == agent.mgmt_ip


@pytest.mark.parametrize("agent, server", [('ubuntu/18.04', 'centos/8')])
def test_search_leases(agent, server):
    # Install Kea on the machine with a Stork Agent.
    agent.install_kea()
    agent.install_kea('kea-dhcp6')
    agent.stop_kea('kea-dhcp4')
    agent.stop_kea('kea-dhcp6')

    # Generate DHCPv4 lease file.
    subprocess.run('./gen-lease4-file.py > kea-leases4.csv', shell=True, check=True)
    agent.upload('kea-leases4.csv', '/var/lib/kea/kea-leases4.csv')
    subprocess.run('rm -rf kea-leases4.csv', shell=True)

    # Generate DHCPv6 lease file.
    subprocess.run('./gen-lease6-file.py > kea-leases6.csv', shell=True, check=True)
    agent.upload('kea-leases6.csv', '/var/lib/kea/kea-leases6.csv')
    subprocess.run('rm -rf kea-leases6.csv', shell=True)

    # Replace DHCPv4 config file.
    agent.upload('kea-dhcp4.conf', '/etc/kea/kea-dhcp4.conf')
    agent.start_kea('kea-dhcp4')

    # Replace DHCPv6 config file.
    agent.upload('kea-dhcp6.conf', '/etc/kea/kea-dhcp6.conf')
    agent.start_kea('kea-dhcp6')

    # login
    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    # Approve agent registration.
    m = _get_machines_and_authorize_them(server)[0]
    assert m['address'] == agent.mgmt_ip

    # Search by IPv4 address..
    leases, conflicts = _search_leases(server, '192.0.2.1')
    assert len(leases) is 1
    assert 'ipAddress' in leases[0]
    assert leases[0]['ipAddress'] == '192.0.2.1'
    assert conflicts is None

    # Search by IPv6 address.
    leases, conflicts = _search_leases(server, '3001:db8:1::1')
    assert len(leases) is 1
    assert 'ipAddress' in leases[0]
    assert leases[0]['ipAddress'] == '3001:db8:1::1'
    assert conflicts is None

    # Search by MAC.
    leases, conflicts = _search_leases(server, '00:01:02:03:04:02')
    assert len(leases) is 1
    assert 'ipAddress' in leases[0]
    assert leases[0]['ipAddress'] == '192.0.2.2'
    assert conflicts is None

    # Search by client id and DUID.
    leases, conflicts = _search_leases(server, '01:02:03:04')
    assert len(leases) is 2
    assert 'ipAddress' in leases[0]
    assert leases[0]['ipAddress'] == '192.0.2.4'
    assert 'ipAddress' in leases[1]
    assert leases[1]['ipAddress'] == '3001:db8:1::4'
    assert conflicts is None

    # Search by hostname.
    leases, conflicts = _search_leases(server, 'host-6.example.org')
    assert len(leases) is 2
    assert 'ipAddress' in leases[0]
    assert leases[0]['ipAddress'] == '192.0.2.6'
    assert 'ipAddress' in leases[1]
    assert leases[1]['ipAddress'] == '3001:db8:1::6'
    assert conflicts is None

    # Search declined leases.
    leases, conflicts = _search_leases(server, 'state:declined')
    assert len(leases) is 20
    for lease in leases:
        # Declined leases should lack identifiers.
        assert 'hwaddr' not in lease
        assert 'clientId' not in lease
        assert 'duid' not in lease
        # The state is declined.
        assert lease['state'] is 1
        # An address should be set.
        assert 'ipAddress' in lease
    # Sanity check addresses returned.
    assert leases[0]['ipAddress'] == '192.0.2.1'
    assert leases[10]['ipAddress'] == '3001:db8:1::1'
    assert conflicts is None

    # Blank search text should return none leases
    r = server.api_get('/leases?text=')
    data = r.json()
    assert data['items'] is None

    r = server.api_get('/leases')
    data = r.json()
    assert data['items'] is None


def run_perfdhcp(src_cntr, dest_ip_addr_or_interf, *, family=4, mac_prefix='00:00', option=None, duid_prefix=None):
    '''
    Run the perfdhcp with provided parameters.

    Positional arguments:
        - dest_ip_addr_or_interf: str
            The IP address or interface name.
            Use the IP address for IPv4 networks and the
            interface name for IPv6 else this tool doesn't work
            due to the unknown reason.

    Keyword arguments:
        - family: int
            The IP family. 4 - IPv4, 6 - IPv6
        - mac_prefix: string or None
            The base mac address prefix, 5 characters (middle should be a colon),
            prefer to use with IPv4. None value disables this feature.
        - option: tuple of two strings or None
            Option to be added to the packets. First item is option ID, second is option value,
            None value disables this feature.
        - duid_prefix: string or None
            The base DUID prefix, 4 digits, prefer to use with IPv6.
            None value disables this feature. Current perfdhcp has probably a bug
            and this flag is ignored.
    '''
    flags = []
    if mac_prefix is not None:
        flags.append("-b mac=" + mac_prefix + ":00:00:00:00")
    if option is not None:
        flags.append("-o %s,%s" % option)
    if duid_prefix is not None:
        flags.append("-b duid=" + duid_prefix + "00000000")
    
    flags_str = " ".join(flags)

    cmd = '/usr/sbin/perfdhcp -%d -r %d -R %d -p 10 %s ' % (family, 10, 10000, flags_str)
    # Is IPv4 or IPv6?
    if '.' not in dest_ip_addr_or_interf and ':' not in dest_ip_addr_or_interf:
        cmd += '-l '
    cmd += dest_ip_addr_or_interf
    result = src_cntr.run(cmd, ignore_error=True)
    if result[0] not in [0, 3]:
        raise Exception('perfdhcp erred: %s' % str(result))


@pytest.mark.parametrize("agent_kea, agent_old_kea, server", [('ubuntu/20.04', 'ubuntu/18.04', 'ubuntu/18.04')])
def test_get_kea_stats(agent_kea, agent_old_kea, server):
    """Check if collecting stats from various Kea versions works.
       DHCPv4 traffic is send to old Kea, then to new kea
       and then it is checked if Stork Server has collected
       stats about the traffic."""
    elems = [(agent_kea, KEA_LATEST, agent_old_kea.mgmt_ip),
             (agent_old_kea, KEA_1_6, agent_kea.mgmt_ip)]
    for idx, (a, ver, relay_addr) in enumerate(elems):
        a.install_kea(kea_version=ver)
        a.install_kea('kea-dhcp6', kea_version=ver)
        a.upload('kea-dhcp4.conf', '/etc/kea/kea-dhcp4.conf')
        a.upload('kea-dhcp6.conf', '/etc/kea/kea-dhcp6.conf')
        # set proper relay address
        a.run(r'sed -i -e s/172\.100\.0\.200/%s/g /etc/kea/kea-dhcp4.conf' % relay_addr)
        # differentiate subnets which are used in this test
        # IPv4
        prefix = '192.%d.2' % idx
        a.run('sed -i -e s/192.0.2/%s/g /etc/kea/kea-dhcp4.conf' % prefix)
        a.run('systemctl restart isc-kea-dhcp4-server')
        # IPv6
        prefix = ':db%d:' % (idx + 8)
        a.run('sed -i -e s/:db8:/%s/g /etc/kea/kea-dhcp6.conf' % prefix)
        a.run('systemctl restart isc-kea-dhcp6-server')

    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    banner("AUTHORIZE MACHINES")
    # get machines (kea and old_kea) that automatically registered in the server and authorize them
    machines = _get_machines_and_authorize_them(server, 2)
    m_new = None
    m_old = None
    for m in machines:
        if m['address'] == agent_kea.mgmt_ip:
            m_new = m
        elif m['address'] == agent_old_kea.mgmt_ip:
            m_old = m
    assert m_new is not None
    assert m_old is not None

    # TODO: CA sometimes does not receive reqs, but works after restart
    agent_kea.run('systemctl restart isc-kea-ctrl-agent')
    agent_old_kea.run('systemctl restart isc-kea-ctrl-agent')

    # check machine state with new kea
    latest_ver = KEA_LATEST.split('-')[0]
    m_new = _get_machine_state(server, m_new['id'])
    assert m_new['apps'] is not None
    assert len(m_new['apps']) == 1
    app = m_new['apps'][0]
    assert app['version'] == latest_ver
    assert len(app['accessPoints']) == 1
    assert app['accessPoints'][0]['address'] == '127.0.0.1'

    # check machine state with old kea
    m_old = _get_machine_state(server, m_old['id'])
    assert m_old['apps'] is not None
    assert len(m_old['apps']) == 1
    app = m_old['apps'][0]
    assert app['version'] == '1.6.3'
    assert len(app['accessPoints']) == 1
    assert app['accessPoints'][0]['address'] == '127.0.0.1'

    # send DHCP traffic to Kea apps
    banner("SEND DHCP TRAFFIC TO KEA APPS")

    # send DHCP traffic to old kea
    agent_kea.run('systemctl stop isc-kea-dhcp4-server')
    agent_kea.run('systemctl stop isc-kea-dhcp6-server')
    run_perfdhcp(agent_kea, agent_old_kea.mgmt_ip)
    run_perfdhcp(agent_kea, agent_old_kea.interface, family=6, duid_prefix='3001')
    agent_kea.run('systemctl start isc-kea-dhcp4-server')
    agent_kea.run('systemctl start isc-kea-dhcp6-server')

    # send DHCP traffic to new kea
    agent_old_kea.run('systemctl stop isc-kea-dhcp4-server')
    agent_old_kea.run('systemctl stop isc-kea-dhcp6-server')
    run_perfdhcp(agent_old_kea, agent_kea.mgmt_ip)
    run_perfdhcp(agent_old_kea, agent_kea.interface, family=6, duid_prefix='3001')
    agent_old_kea.run('systemctl start isc-kea-dhcp4-server')
    agent_old_kea.run('systemctl start isc-kea-dhcp6-server')

    # check gathered stats by Stork server
    for i in range(80):
        r = server.api_get('/overview')
        data = r.json()
        if data['dhcp4Stats'] and 'assignedAddresses' in data['dhcp4Stats']:
            # sent 100 requests to 2 subnets so there should be 200 leases
            # so at least 150
            if data['dhcp4Stats']['assignedAddresses'] > 150:
                break
        time.sleep(2)

    assert data['dhcp4Stats']
    assert 'assignedAddresses' in data['dhcp4Stats']
    assert 'assignedNAs' in data['dhcp6Stats']
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


@pytest.mark.parametrize("agent, server, bind_version", [('centos/8', 'ubuntu/18.04', None),
                                                         ('centos/8', 'ubuntu/18.04', '9.11'),
                                                         ('centos/8', 'ubuntu/18.04', '9.16'),
                                                         ('centos/8', 'ubuntu/18.04', '9.17'),
                                                         ('centos/8', 'ubuntu/18.04', None),
                                                         ('ubuntu/18.04', 'centos/8', None),
                                                         ('ubuntu/18.04', 'centos/8', '9.11'),
                                                         ('ubuntu/18.04', 'centos/8', '9.16'),
                                                         ('ubuntu/18.04', 'centos/8', '9.17'),
                                                         ('ubuntu/20.04', 'centos/8', None)])
def test_bind9_versions(agent, server, bind_version):
    """Check if Stork agent detects different BIND 9 versions."""
    # install kea on the agent machine
    agent.install_bind(bind_version=bind_version)

    # login
    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    # get machine that automatically registered in the server and authorize it
    m = _get_machines_and_authorize_them(server)[0]

    # check machine state
    m = _get_machine_state(server, m['id'])
    assert m['apps'] is not None
    assert len(m['apps']) == 1
    if bind_version:
        assert bind_version in m['apps'][0]['version']
    assert len(m['apps'][0]['accessPoints']) == 2
    assert m['apps'][0]['accessPoints'][0]['address'] == '127.0.0.1'


@pytest.mark.parametrize("agent, server", [('ubuntu/18.04', 'centos/8')])
def test_get_host_leases(agent, server):
    # Install Kea on the machine with a Stork Agent.
    agent.install_kea()
    agent.install_kea('kea-dhcp6')
    agent.stop_kea('kea-dhcp4')
    agent.stop_kea('kea-dhcp6')

    # Generate DHCPv4 lease file.
    subprocess.run('./gen-lease4-file.py > kea-leases4.csv', shell=True, check=True)
    agent.upload('kea-leases4.csv', '/var/lib/kea/kea-leases4.csv')
    subprocess.run('rm -rf kea-leases4.csv', shell=True)

    # Generate DHCPv6 lease file.
    subprocess.run('./gen-lease6-file.py > kea-leases6.csv', shell=True, check=True)
    agent.upload('kea-leases6.csv', '/var/lib/kea/kea-leases6.csv')
    subprocess.run('rm -rf kea-leases6.csv', shell=True)

    # Replace DHCPv4 config file.
    agent.upload('kea-dhcp4.conf', '/etc/kea/kea-dhcp4.conf')
    agent.start_kea('kea-dhcp4')

    # Replace DHCPv6 config file.
    agent.upload('kea-dhcp6.conf', '/etc/kea/kea-dhcp6.conf')
    agent.start_kea('kea-dhcp6')

    # login
    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    # Approve agent registration.
    _get_machines_and_authorize_them(server)

    # Find the host reservation for which we will be checking the
    # leases status.
    host_id = 0
    for i in range(10):
        r = server.api_get('/hosts?text=192.0.2.2')
        data = r.json()

        if 'items' in data and data['items'] and len(data['items']) > 0:
            # Record host id, which will be later used to search leases.
            host_id = data['items'][0]['id']
            break
        time.sleep(2)
    assert host_id > 0, 'failed to find host with IP address `%s`' % '192.0.2.2'

    # Find the leases for the host.
    leases, conflicts = _search_leases(server, host_id=host_id)
    assert len(leases) is 1
    assert 'ipAddress' in leases[0]
    assert leases[0]['ipAddress'] == '192.0.2.2'
    assert conflicts is None

    # Find the host reservation for IPv6 address.
    host_id = 0
    for i in range(10):
        r = server.api_get('/hosts?text=3001:db8:1::2')
        data = r.json()

        if 'items' in data and data['items'] and len(data['items']) > 0:
            # Record host id, which will be later used to search leases.
            host_id = data['items'][0]['id']
            break
        time.sleep(2)
    assert host_id > 0, 'failed to find host with IP address `%s`' % '3001:db8:1::2'

    # Find leases for the IPv6 host reservation.
    leases, conflicts = _search_leases(server, host_id=host_id)
    assert len(leases) == 1
    assert 'ipAddress' in leases[0]
    assert leases[0]['ipAddress'] == '3001:db8:1::2'

    # The lease was assigned to a different client. There should be
    # a conflict returned.
    assert conflicts is not None
    assert len(conflicts) is 1
    assert conflicts[0] is leases[0]['id']

    # Find leases for non-existing host id.
    r = server.api_get('/leases?hostId=1000000')
    data = r.json()
    assert data['items'] is None


@pytest.mark.parametrize("agent, server", [('ubuntu/18.04', 'ubuntu/18.04')])
def test_agent_reregistration_after_restart(agent, server):
    """Check if after restart the agent isn't re-register.
       It should use the same agent token and certs as before restart."""

    agent_cert_files = [
        '/var/lib/stork-agent/certs/key.pem',
        '/var/lib/stork-agent/certs/cert.pem',
        '/var/lib/stork-agent/certs/ca.pem',
        '/var/lib/stork-agent/tokens/agent-token.txt'
    ]

    # install kea on the agent machine
    agent.install_kea()

    # prepare kea config
    banner("UPLOAD KEA CONFIG")
    agent.upload('kea-dhcp4.conf', '/etc/kea/kea-dhcp4.conf')
    agent.run('systemctl restart isc-kea-dhcp4-server')

    # login
    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    # get machine that automatically registered in the server and authorize it
    m = _get_machines_and_authorize_them(server)[0]
    assert m['address'] == agent.mgmt_ip

    # get agent token
    agent_token_before = m['agentToken']

    # get agent cert file hashes
    hashes_before = [agent.run('sha1sum %s' % p).stdout.split()[0] for p in agent_cert_files]

    # restart agent
    agent.run('systemctl restart isc-kea-ctrl-agent')

    # get machine after restart
    m = _get_machines(server)[0]

    # get agent token after restart
    agent_token_after = m['agentToken']

    # get agent cert file hashes after restart
    hashes_after = [agent.run('sha1sum %s' % p).stdout.split()[0] for p in agent_cert_files]

    # the agent token and cert files should be the same as before restart
    assert agent_token_before == agent_token_after
    assert tuple(hashes_before) == tuple(hashes_after)

@pytest.mark.parametrize("agent, server", [('ubuntu/18.04', 'ubuntu/18.04')])
def test_agent_over_ipv6(agent, server):
    # Setup the Stork Agent over IPv6
    agent.set_stork_agent_ip6_address()
    agent.install_kea()
    
    # login
    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    # get machine that automatically registered in the server and authorize it
    # but only machines that use IPv6
    m = _get_machines_and_authorize_them(server,
        machine_filter=lambda m: ':' in m['address'])[0]
    assert m['address'] == agent.mgmt_ip6

    m = _get_machine_state(server, m['id'])
    assert m["apps"] is not None
    assert len(m["apps"]) > 0


@pytest.mark.parametrize("agent, server", [('centos/8', 'ubuntu/18.04')])
def test_communication_with_kea_over_secure_protocol(agent, server):
    """Check if Stork agent communicates with Kea over HTTPS correctly.
    In this test the Kea doesn't require SSL certificate on the client side."""
    # install kea on the agent machine
    agent.set_skip_tls_cert_verification()
    agent.install_kea(tls_support=KeaTLSSupport.OPTIONAL_CLIENT_CERT)

    # login
    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    # get machine that automatically registered in the server and authorize it
    m = _get_machines_and_authorize_them(server)[0]
    # trig forward command to Kea over HTTPS
    m = _get_machine_state(server, m['id'])

    assert m['address'] == agent.mgmt_ip
    assert m['apps'][0]['accessPoints'][0]['useSecureProtocol']

    # Stork Agent should correctly connect to the Kea CA.
    # No error can occur.
    _wait_for_event(server, 'Failed to forward commands to Kea', expected=False)


@pytest.mark.parametrize("agent, server", [('centos/8', 'ubuntu/18.04')])
def test_communication_with_kea_over_secure_protocol_nontrusted_client(agent: containers.StorkAgentContainer, server):
    """The Stork Agent uses self-signed TLS certificates over HTTPS, but the Kea
    requires the valid credentials. The Stork Agent should send request, but get
    rejection from the Kea CA."""
    # install kea on the agent machine
    agent.set_skip_tls_cert_verification()
    agent.install_kea(tls_support=KeaTLSSupport.REQUIRE_CLIENT_CERT)

    # login
    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    # get machine that automatically registered in the server and authorize it
    m = _get_machines_and_authorize_them(server)[0]
    # trig forward command to Kea over HTTPS
    m = _get_machine_state(server, m['id'])

    assert m['address'] == agent.mgmt_ip
    assert m['apps'][0]['accessPoints'][0]['useSecureProtocol']

    ca_svc_name = 'kea-ctrl-agent' if 'centos' in agent.name else 'isc-kea-ctrl-agent'
    res = agent.run('journalctl -xeu ' + ca_svc_name)
    assert 'TLS handshake with 127.0.0.1 failed with certificate verify failed' in res.stdout


@pytest.mark.parametrize("agent, server", [('centos/8', 'ubuntu/18.04')])
def test_communication_with_kea_over_secure_protocol_require_trusted_cert(agent, server):
    """Check if Stork agent requires a trusted Kea cert if specific flag is not set.
    In this test the Kea doesn't require TLS certificate on the client side."""
    # install kea on the agent machine
    agent.install_kea(tls_support=KeaTLSSupport.OPTIONAL_CLIENT_CERT)

    # login
    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    # get machine that automatically registered in the server and authorize it
    m = _get_machines_and_authorize_them(server)[0]
    # trig forward command to Kea over HTTPS
    m = _get_machine_state(server, m['id'])

    assert m['address'] == agent.mgmt_ip
    assert m['apps'][0]['accessPoints'][0]['useSecureProtocol']

    ca_svc_name = 'kea-ctrl-agent' if 'centos' in agent.name else 'isc-kea-ctrl-agent'
    res = agent.run('journalctl -xeu ' + ca_svc_name)
    assert 'TLS handshake with 127.0.0.1 failed with sslv3 alert bad certificate' in res.stdout

@pytest.mark.parametrize("agent, server", [('ubuntu/18.04', 'centos/8')])
def test_get_dhcp4_config_review_reports(agent, server):
    """Test that the Stork server performs Kea configuration review and returns the reports."""
    # Install kea on the agent machine.
    agent.install_kea()

    # Get Kea config having issues which Stork configuration review should catch.
    banner("UPLOAD KEA CONFIG")
    agent.upload('kea-dhcp4-config-review.conf', '/etc/kea/kea-dhcp4.conf')
    agent.run('systemctl restart isc-kea-dhcp4-server')

    # login
    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    # Get the machine that automatically registered in the server and authorize it.
    m = _get_machines_and_authorize_them(server)[0]
    assert m['address'] == agent.mgmt_ip

    # Check machine state.
    m = _get_machine_state(server, m['id'])

    # Retrieve daemon id.
    assert m['apps'] is not None
    assert len(m['apps']) == 1
    daemons = m['apps'][0]['details']['daemons']
    daemons = [d for d in daemons if d['name'] == 'dhcp4']
    assert len(daemons) == 1
    daemon_id = daemons[0]['id']

    # Get config reports for the daemon.
    banner("GET CONFIG REPORTS")
    r = server.api_get('/daemons/%d/config-reports?start=0&limit=10' % daemon_id)
    data = r.json()

    # Expecting one report indicating that the stat_cmds hooks library
    # was not loaded.
    assert data['total'] is not None
    assert data['total'] == 1
    assert data['items'] is not None
    assert len(data['items']) == 1
    assert data['items'][0]['checker'] is not None
    assert data['items'][0]['checker'] == 'stat_cmds_presence'

@pytest.mark.parametrize("agent, server", [('centos/8', 'ubuntu/18.04')])
def test_communication_with_kea_using_basic_auth_no_credentials(agent: containers.StorkAgentContainer, server):
    """The Kea CA is configured to accept requests only with Basic Auth credentials in header.
    The Stork Agent doesn't have a credentials file. Kea shouldn't accept the requests from the Stork Agent."""
    # install kea on the agent machine
    agent.set_skip_tls_cert_verification()
    agent.install_kea(basic_auth_enable=True)

    # login
    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    # get machine that automatically registered in the server and authorize it
    m = _get_machines_and_authorize_them(server)[0]
    # trig forward command to Kea
    m = _get_machine_state(server, m['id'])

    # The Stork Agent doesn't know the credentials yet.
    # The above request should fail.
    _wait_for_event(server, 'communication with CA daemon of <app id="0" name="" type="kea" version=""> failed',
        details='"result": 401, "text": "Unauthorized"')


@pytest.mark.parametrize("agent, server", [('centos/8', 'ubuntu/18.04')])
def test_communication_with_kea_using_basic_auth(agent: containers.StorkAgentContainer, server):
    """The Kea CA is configured to accept requests only with Basic Auth credentials in header.
    The Stork Agent has a credentials file. Kea should accept the requests from the Stork Agent."""
    # Reconfigure the agent to use Basic Auth credentials
    agent.use_credentials_file()
    # install kea on the agent machine
    agent.set_skip_tls_cert_verification()
    agent.install_kea(basic_auth_enable=True)

    # login
    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)  # TODO: POST should return 201
    assert r.json()['login'] == 'admin'

    # get machine that automatically registered in the server and authorize it
    m = _get_machines_and_authorize_them(server)[0]
    # trig forward command to Kea
    m = _get_machine_state(server, m['id'])

    _wait_for_event(server, 'communication with CA daemon of <app id="0" name="" type="kea" version=""> failed',
        details='"result": 401, "text": "Unauthorized"', expected=False)

@pytest.mark.parametrize("server", [('ubuntu/18.04')])
def test_server_enable_sslmode(server):
    server.enable_database_ssl()
    server.enable_sslmode('require')
    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)
    assert r.json()['login'] == 'admin'

    server.enable_sslmode('verify-ca')
    r = server.api_post('/sessions', json=dict(useremail='admin', userpassword='admin'), expected_status=200)
    assert r.json()['login'] == 'admin'
