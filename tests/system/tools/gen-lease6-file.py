#!/usr/bin/env python3

import csv
import sys
import time

def main():
    # DHCPv6 lease file header.
    fileheader =[
        'address',
        'duid',
        'valid_lifetime',
        'expire',
        'subnet_id',
        'pref_lifetime',
        'lease_type',
        'iaid',
        'prefix_len',
        'fqdn_fwd',
        'fqdn_rev',
        'hostname',
        'hwaddr',
        'state',
        'user_context'
    ]
    # Create CSV writer and output its header.
    lease_writer = csv.DictWriter(sys.stdout, fieldnames=fileheader, lineterminator='\n')
    lease_writer.writeheader()

    # Generate some leases and output them to the lease file.
    for i in range(1, 20):
        # Even leases are in default state. Odd leases are declined.
        lease = {
            'duid': '0',
            'valid_lifetime': 600,
            'subnet_id': 1,
            'pref_lifetime': 300,
            'lease_type': 0,
            'iaid': 7,
            'prefix_len': 128,
            'fqdn_fwd': 1,
            'fqdn_rev': 1,
            'hostname': 'host-%d.example.org' % i,
            'state': i % 2
        }
        lease['address'] = '3001:db8:1::%d' % i
        lease['expire'] = '%d' % (time.time() + 600)

        # Only non-declined leases contain MAC address and client id.
        if i % 2 == 0:
            lease['duid'] = '01:02:03:%02d' % i

        lease_writer.writerow(lease)

if __name__ == "__main__":
    main()
