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


def main():
    global app
    app = Flask(__name__, static_url_path='', static_folder='')

    url = 'http://server:8080/api/subnets?start=0&limit=100'
    r = requests.get(url)
    data = r.json()

    for sn in data['items']:
        sn['rate'] = 1
        sn['clients'] = 1000
        sn['state'] = 'stop'
        sn['proc'] = None

    app.subnets = data

main()


@app.route('/')
def root():
    return app.send_static_file('index.html')


@app.route('/subnets')
def get_subnets():
    return serialize_subnets(app.subnets)


def serialize_subnets(subnets):
    data = dict(total=subnets['total'], items=[])
    for sn in subnets['items']:
        data['items'].append(dict(subnet=sn['subnet'],
                                  rate=sn['rate'],
                                  clients=sn['clients'],
                                  state=sn['state']))
    return json.dumps(data)


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

        # start perfdhcp if requested
        elif subnet['state'] == 'stop' and data['state'] == 'start':
            subnet['proc'] = start_perfdhcp(subnet)

        subnet['state'] = data['state']

    return serialize_subnets(app.subnets)
