import os
import sys
import json
import shlex
import pprint
import subprocess
from xmlrpc.client import ServerProxy

from flask import Flask, escape, request, send_from_directory
import requests

app = None

STORK_SERVER_URL = os.environ.get('STORK_SERVER_URL', 'http://server:8080')


def _login_session():
    s = requests.Session()
    credentials = dict(
        useremail='admin',
        userpassword='admin'
    )
    s.post('%s/api/sessions' % STORK_SERVER_URL, json=credentials)
    return s


def start_perfdhcp(subnet):
    rate, clients = subnet['rate'], subnet['clients']
    client_class = subnet['clientClass']
    mac_prefix = client_class[6:].replace('-', ':')
    mac_prefix_bytes = mac_prefix.split(':')
    kea_addr = '172.1%s.0.100' % mac_prefix_bytes[0]
    cmd = '/usr/sbin/perfdhcp -4 -r %d -R %d -b mac=%s:00:00:00:00 %s' % (rate, clients, mac_prefix, kea_addr)
    args = shlex.split(cmd)
    print('exec: %s' % cmd, file=sys.stderr)
    return subprocess.Popen(args)


def _refresh_subnets():
    try:
        app.subnets = dict(items=[], total=0)

        s = _login_session()

        url = '%s/api/subnets?start=0&limit=100' % STORK_SERVER_URL
        r = s.get(url)
        data = r.json()
        app.logger.info('SN %s', data)

        if not data:
            return

        for sn in data['items']:
            sn['rate'] = 1
            sn['clients'] = 1000
            sn['state'] = 'stop'
            sn['proc'] = None
            if 'sharedNetwork' not in sn:
                sn['sharedNetwork'] = ''

        app.subnets = data
    except Exception as e:
        app.logger.info("IGNORED EXCEPTION %s" % str(e))


def serialize_subnets(subnets):
    data = dict(total=subnets['total'], items=[])
    for sn in subnets['items']:
        data['items'].append(dict(subnet=sn['subnet'],
                                  sharedNetwork=sn['sharedNetwork'],
                                  rate=sn['rate'],
                                  clients=sn['clients'],
                                  state=sn['state']))
    return json.dumps(data)


def run_dig(server):
    clients = server['clients']
    qname = server['qname']
    qtype = server['qtype']
    tcp = "+notcp"
    if server['transport'] == 'tcp':
        tcp = "+tcp"
    address = server['machine']['address']
    cmd = 'dig %s +tries=1 +retry=0 @%s %s %s' % (tcp, address, qname, qtype)
    print('exec %d times: %s' % (clients, cmd), file=sys.stderr)
    for i in range(0, clients):
        args = shlex.split(cmd)
        subprocess.run(args)


def start_flamethrower(server):
    rate = server['rate']*1000
    clients = server['clients']
    qname = server['qname']
    qtype = server['qtype']
    transport = "udp"
    if server['transport'] == 'tcp':
        transport = "tcp"
    address = server['machine']['address']
    # send one query (-q) per client (-c) every 'rate' millisecond (-d)
    # on transport (-P) with qname (-r) and qtype (-T)
    cmd = 'flame -q 1 -c %d -d %d -P %s -r %s -T %s %s' % (clients, rate, transport, qname, qtype, address)
    args = shlex.split(cmd)
    print('exec: %s' % cmd, file=sys.stderr)
    return subprocess.Popen(args)


def _refresh_servers():
    try:
        app.servers = dict(items=[], total=0)

        s = _login_session()

        url = '%s/api/apps/' % STORK_SERVER_URL
        r = s.get(url)
        data = r.json()

        if not data:
            return

        for srv in data['items']:
            if srv['type'] == 'bind9':
                srv['clients'] = 1
                srv['rate'] = 1
                srv['qname'] = 'example.com'
                srv['qtype'] = 'A'
                srv['transport'] = 'udp'
                srv['proc'] = None
                srv['state'] = 'stop'
                app.servers['items'].append(srv)

        print('data: %s' % app.servers, file=sys.stderr)

    except Exception as e:
        app.logger.info("IGNORED EXCEPTION %s" % str(e))


def serialize_servers(servers):
    data = dict(total=servers['total'], items=[])
    for srv in servers['items']:
        data['items'].append(dict(state=srv['state'],
                                  address=srv['machine']['address'],
                                  clients=srv['clients'],
                                  rate=srv['rate'],
                                  transport=srv['transport'],
                                  qtype=srv['qtype'],
                                  qname=srv['qname']))
    return json.dumps(data)


def main():
    global app
    app = Flask(__name__, static_url_path='', static_folder='')

    _refresh_subnets()
    _refresh_servers()


main()


@app.route('/')
def root():
    return app.send_static_file('index.html')


@app.route('/subnets')
def get_subnets():
    _refresh_subnets()
    return serialize_subnets(app.subnets)


@app.route('/subnets/<int:index>', methods=['PUT'])
def put_subnet_params(index):
    data = json.loads(request.data)
    subnet = app.subnets['items'][index]

    if 'rate' in data:
        subnet['rate'] = data['rate']

    if 'clients' in data:
        subnet['clients'] = data['clients']

    if 'state' in data:
        # stop perfdhcp if requested
        if subnet['state'] == 'start' and data['state'] == 'stop' and subnet['proc'] is not None:
            subnet['proc'].terminate()
            subnet['proc'].wait()
            subnet['proc'] = None

        # start perfdhcp if requested but if another subnet in the same shared network is running
        # then stop it first
        elif subnet['state'] == 'stop' and data['state'] == 'start':
            if subnet['sharedNetwork'] != '':
                for sn in app.subnets['items']:
                    if sn['sharedNetwork'] == subnet['sharedNetwork'] and sn['state'] == 'start':
                        sn['proc'].terminate()
                        sn['proc'].wait()
                        sn['proc'] = None
                        sn['state'] = 'stop'

            subnet['proc'] = start_perfdhcp(subnet)

        subnet['state'] = data['state']

    return serialize_subnets(app.subnets)


@app.route('/servers')
def get_servers():
    _refresh_servers()
    return serialize_servers(app.servers)


@app.route('/query/<int:index>', methods=['PUT'])
def put_query_params(index):
    data = json.loads(request.data)
    server = app.servers['items'][index]

    if 'qname' in data:
        server['qname'] = data['qname']

    if 'qtype' in data:
        server['qtype'] = data['qtype']

    if 'transport' in data:
        server['transport'] = data['transport']

    if 'clients' in data:
        server['clients'] = data['clients']

    if 'rate' in data:
        server['rate'] = data['rate']

    run_dig(server)

    return serialize_servers(app.servers)


@app.route('/perf/<int:index>', methods=['PUT'])
def put_perf_params(index):
    data = json.loads(request.data)
    server = app.servers['items'][index]

    if 'qname' in data:
        server['qname'] = data['qname']

    if 'qtype' in data:
        server['qtype'] = data['qtype']

    if 'transport' in data:
        server['transport'] = data['transport']

    if 'clients' in data:
        server['clients'] = data['clients']

    if 'rate' in data:
        server['rate'] = data['rate']

    if 'state' in data:
        # stop dnsperf if requested
        if server['state'] == 'start' and data['state'] == 'stop' and server['proc'] is not None:
            server['proc'].terminate()
            server['proc'].wait()
            server['proc'] = None

        # start dnsperf if requested
        if server['state'] == 'stop' and data['state'] == 'start':
            server['proc'] = start_flamethrower(server)

        server['state'] = data['state']

    return serialize_servers(app.servers)


def _get_services():
    app.services = dict(items=[], total=0)

    s = _login_session()

    url = '%s/api/machines?start=0&limit=100' % STORK_SERVER_URL
    r = s.get(url)
    machines = r.json()['items']
    if machines is None:
        machines = []

    data = dict(total=0, items=[])
    for m in machines:
        s = ServerProxy('http://%s:9001/RPC2' % m['address'])
        try:
            services = s.supervisor.getAllProcessInfo()
        except:
            continue
        pprint.pprint(services)
        for srv in services:
            srv['machine'] = m['address']
            data['items'].append(srv)

    data['total'] = len(data['items'])

    app.services = data

    return data


@app.route('/services')
def get_services():
    data = _get_services()
    return json.dumps(data)


@app.route('/services/<int:index>', methods=['PUT'])
def put_service(index):
    data = json.loads(request.data)
    service = app.services['items'][index]

    s = ServerProxy('http://%s:9001/RPC2' % service['machine'])

    if data['operation'] == 'stop':
        s.supervisor.stopProcess(service['name'])
    elif data['operation'] == 'start':
        s.supervisor.startProcess(service['name'])

    data = _get_services()
    return json.dumps(data)
