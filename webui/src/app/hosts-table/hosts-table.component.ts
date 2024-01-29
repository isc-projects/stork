import { Component, OnInit } from '@angular/core'

export interface Host {
    mac_address
    ip_address
}

@Component({
    selector: 'app-hosts-table',
    templateUrl: './hosts-table.component.html',
    styleUrls: ['./hosts-table.component.sass'],
})
export class HostsTableComponent implements OnInit {
    hosts: Host[]

    constructor() {}

    ngOnInit() {
        this.hosts = []
        for (let i = 0; i < 10; i++) {
            const mac = []
            for (let j = 0; j < 6; j++) {
                const n = Math.floor(Math.random() * 99)
                mac.push(n)
            }
            const ip = []
            for (let j = 0; j < 4; j++) {
                const n = Math.floor(Math.random() * 256)
                ip.push(n)
            }
            const rec = {
                mac_address: mac.join(':'),
                ip_address: ip.join('.'),
            }
            this.hosts.push(rec)
        }
    }
}
