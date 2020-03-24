import sys
import json
import shlex
import subprocess

from flask import Flask, escape, request, send_from_directory
import requests

app = None

def run_dig(server):
    clients = server['clients']
    qname = server['qname']
    address = server['machine']['address']
    cmd = 'dig +tries=1 +retry=0 @%s %s' % (address, qname)
    print('exec %d times: %s' % (clients, cmd), file=sys.stderr)
    for i in range(0, clients):
        args = shlex.split(cmd)
        subprocess.run(args)


def start_flamethrower(server):
    rate = server['rate']*1000
    clients = server['clients']
    qname = server['qname']
    address = server['machine']['address']
    # send one query (-q) per client (-c) every 'rate' millisecond (-d)
    cmd = 'flame -q 1 -c %d -d %d -r %s %s' % (clients, rate, qname, address)
    args = shlex.split(cmd)
    print('exec: %s' % cmd, file=sys.stderr)
    return subprocess.Popen(args)


def _refresh_servers():
    try:
        app.servers = dict(items=[], total=0)

        url = 'http://server:8080/api/apps/'
        r = requests.get(url)
        data = r.json()

        if not data:
            return

        for srv in data['items']:
            if srv['type'] == 'bind9':
                srv['clients'] = 1
                srv['rate'] = 1
                srv['qname'] = 'example.com'
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
                                  qname=srv['qname']))
    return json.dumps(data)


def main():
    global app
    app = Flask(__name__, static_url_path='', static_folder='')

    _refresh_servers()


main()


@app.route('/')
def root():
    return app.send_static_file('index.html')

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

    if 'clients' in data:
        server['clients'] = data['clients']

    run_dig(server)

    return serialize_servers(app.servers)

@app.route('/perf/<int:index>', methods=['PUT'])
def put_perf_params(index):
    data = json.loads(request.data)
    server = app.servers['items'][index]

    if 'qname' in data:
        server['qname'] = data['qname']

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
