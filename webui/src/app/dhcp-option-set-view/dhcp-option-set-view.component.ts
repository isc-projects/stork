import { Component, Input, OnInit } from '@angular/core'
import { TreeNode } from 'primeng/api'
import { DHCPOption } from '../backend/model/dHCPOption'
import { Host } from '../backend/model/host'

interface Node {}

@Component({
    selector: 'app-dhcp-option-set-view',
    templateUrl: './dhcp-option-set-view.component.html',
    styleUrls: ['./dhcp-option-set-view.component.sass'],
})
export class DhcpOptionSetViewComponent implements OnInit {
    @Input() host: Host

    optionNodes: TreeNode[]

    constructor() {
        this.optionNodes = [
            {
                key: '53',
                type: 'option',
                expanded: true,
                data: {
                    alwaysSend: true,
                    code: 53,
                },
                children: [
                    {
                        type: 'fields',
                        expanded: true,
                        data: [
                            {
                                value: '111',
                            },
                            {
                                value: 'ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff',
                            },
                            {
                                value: 'www.example.example.example.example.example.example.example.example.com',
                            },
                        ],
                    },
                    {
                        key: '53-1',
                        type: 'suboption',
                        expanded: true,
                        data: {
                            code: 1,
                        },
                        children: [
                            {
                                key: '53-1',
                                expanded: true,
                                type: 'fields',
                                data: [
                                    {
                                        value: 123,
                                    },
                                ],
                            },
                        ],
                    },
                ],
            },
        ]
    }

    ngOnInit(): void {}
}
