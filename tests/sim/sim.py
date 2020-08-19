import sys
import json
import shlex
import subprocess

from flask import Flask, escape, request, send_from_directory
import requests

app = None

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

        s = requests.Session()
        credentials = dict(
            useremail='admin',
            userpassword='admin'
        )
        s.post('http://server:8080/api/sessions', json=credentials)

        url = 'http://server:8080/api/subnets?start=0&limit=100'
        r = s.get(url)
        data = r.json()

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


def main():
    global app
    app = Flask(__name__, static_url_path='', static_folder='')

    _refresh_subnets()


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
