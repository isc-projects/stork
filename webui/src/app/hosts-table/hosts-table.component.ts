import { Component, OnInit } from '@angular/core';

import {TableModule} from 'primeng/table';

export interface Host {
    mac_address;
    ip_address;
}

@Component({
  selector: 'hosts-table',
  templateUrl: './hosts-table.component.html',
  styleUrls: ['./hosts-table.component.sass']
})
export class HostsTableComponent implements OnInit {

    hosts: Host[];

    constructor() { }

    ngOnInit() {
        this.hosts = [];
        for (var i = 0; i < 10; i++) {
            var mac = [];
            for (var j = 0; j < 6; j++) {
                var n = Math.floor(Math.random() * 99);
                mac.push(n);
            }
            var ip = [];
            for (var j = 0; j < 4; j++) {
                var n = Math.floor(Math.random() * 256);
                ip.push(n);
            }
            var rec = {
                mac_address: mac.join(':'),
                ip_address: ip.join('.')
            }
            this.hosts.push(rec);
        }
    }

}
