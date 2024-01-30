import { ComponentFixture, TestBed } from '@angular/core/testing'
import { By } from '@angular/platform-browser'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { TagModule } from 'primeng/tag'
import { TooltipModule } from 'primeng/tooltip'
import { TreeNode } from 'primeng/api'
import { TreeModule } from 'primeng/tree'
import { DhcpOptionSetViewComponent, OptionFieldNode, OptionNode } from './dhcp-option-set-view.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { DHCPOption } from '../backend/model/dHCPOption'
import { DividerModule } from 'primeng/divider'
import { CheckboxModule } from 'primeng/checkbox'
import { FormsModule } from '@angular/forms'
import { IPType } from '../iptype'

describe('DhcpOptionSetViewComponent', () => {
    let component: DhcpOptionSetViewComponent
    let fixture: ComponentFixture<DhcpOptionSetViewComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [
                CheckboxModule,
                DividerModule,
                FormsModule,
                NoopAnimationsModule,
                OverlayPanelModule,
                TagModule,
                TooltipModule,
                TreeModule,
            ],
            declarations: [DhcpOptionSetViewComponent, HelpTipComponent],
        }).compileComponents()
    })

    beforeEach(() => {
        fixture = TestBed.createComponent(DhcpOptionSetViewComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should convert DHCP options to a tree', () => {
        let options: Array<Array<DHCPOption>> = [
            [
                {
                    alwaysSend: true,
                    code: 1024,
                    fields: [
                        {
                            fieldType: 'uint32',
                            values: ['111'],
                        },
                        {
                            fieldType: 'ipv6-prefix',
                            values: ['3000::', '64'],
                        },
                    ],
                    universe: 6,
                    options: [
                        {
                            code: 1025,
                            universe: 6,
                        },
                        {
                            code: 1026,
                            fields: [
                                {
                                    fieldType: 'ipv6-address',
                                    values: ['2001:db8:1::1'],
                                },
                                {
                                    fieldType: 'ipv6-address',
                                    values: ['2001:db8:2::1'],
                                },
                            ],
                            universe: 6,
                        },
                    ],
                },
                {
                    code: 1027,
                    fields: [
                        {
                            fieldType: 'bool',
                            values: ['true'],
                        },
                    ],
                    universe: 6,
                },
                {
                    code: 1028,
                    options: [
                        {
                            code: 1029,
                            fields: [
                                {
                                    fieldType: 'string',
                                    values: ['foo'],
                                },
                            ],
                            options: [
                                {
                                    code: 1030,
                                    options: [
                                        {
                                            code: 1031,
                                        },
                                    ],
                                },
                            ],
                            universe: 6,
                        },
                    ],
                    universe: 6,
                },
            ],
        ]
        component.options = options
        component.levels = ['subnet']
        component.ngOnInit()
        fixture.detectChanges()

        expect(component.optionNodes[0].length).toBe(3)

        // Option 1024.
        expect((component.optionNodes[0][0] as TreeNode<OptionNode>).data.alwaysSend).toBeTrue()
        expect((component.optionNodes[0][0] as TreeNode<OptionNode>).data.code).toBe(1024)

        // Option 1024 fields.
        expect(component.optionNodes[0][0].children.length).toBe(4)
        expect(component.optionNodes[0][0].children[0].type).toBe('field')
        expect(component.optionNodes[0][0].children[0].expanded).toBeTrue()
        expect((component.optionNodes[0][0].children[0] as TreeNode<OptionFieldNode>).data.fieldType).toBe('uint32')
        expect((component.optionNodes[0][0].children[0] as TreeNode<OptionFieldNode>).data.value).toBe('111')
        expect(component.optionNodes[0][0].children[1].type).toBe('field')
        expect(component.optionNodes[0][0].children[1].expanded).toBeTrue()
        expect((component.optionNodes[0][0].children[1] as TreeNode<OptionFieldNode>).data.fieldType).toBe(
            'ipv6-prefix'
        )
        expect((component.optionNodes[0][0].children[1] as TreeNode<OptionFieldNode>).data.value).toBe('3000::/64')

        // Option 1024 suboptions.
        expect((component.optionNodes[0][0].children[2] as TreeNode<OptionNode>).data.code).toBe(1025)
        expect(component.optionNodes[0][0].children[2].children.length).toBe(0)

        // Suboption 1026.
        expect((component.optionNodes[0][0].children[3] as TreeNode<OptionNode>).data.code).toBe(1026)

        // Suboption 1026 fields.
        expect(component.optionNodes[0][0].children[3].children.length).toBe(2)
        expect((component.optionNodes[0][0].children[3].children[0] as TreeNode<OptionFieldNode>).data.fieldType).toBe(
            'ipv6-address'
        )
        expect((component.optionNodes[0][0].children[3].children[0] as TreeNode<OptionFieldNode>).data.value).toBe(
            '2001:db8:1::1'
        )
        expect((component.optionNodes[0][0].children[3].children[1] as TreeNode<OptionFieldNode>).data.fieldType).toBe(
            'ipv6-address'
        )
        expect((component.optionNodes[0][0].children[3].children[1] as TreeNode<OptionFieldNode>).data.value).toBe(
            '2001:db8:2::1'
        )
        // Option 1027.
        expect((component.optionNodes[0][1] as TreeNode<OptionNode>).data.alwaysSend).toBeFalsy()
        expect((component.optionNodes[0][1] as TreeNode<OptionNode>).data.code).toBe(1027)
        expect(component.optionNodes[0][1].children.length).toBe(1)
        expect(component.optionNodes[0][1].children[0].type).toBe('field')
        expect(component.optionNodes[0][1].children[0].expanded).toBeTrue()
        expect((component.optionNodes[0][1].children[0] as TreeNode<OptionFieldNode>).data.fieldType).toBe('bool')
        expect((component.optionNodes[0][1].children[0] as TreeNode<OptionFieldNode>).data.value).toBe('true')

        // Option 1028.
        expect((component.optionNodes[0][2] as TreeNode<OptionNode>).data.alwaysSend).toBeFalsy()
        expect((component.optionNodes[0][2] as TreeNode<OptionNode>).data.code).toBe(1028)
        expect(component.optionNodes[0][2].children.length).toBe(1)

        // Suboption 1029.
        const option1029 = component.optionNodes[0][2].children[0] as TreeNode<OptionNode>
        expect(option1029.data.code).toBe(1029)
        expect(option1029.children.length).toBe(2)

        // Suboption 1030.
        const option1030 = option1029.children[1] as TreeNode<OptionNode>
        expect(option1030.data.code).toBe(1030)
        expect(option1030.children.length).toBe(0)

        // Make sure that appropriate tags are displayed.
        let optionTags = fixture.debugElement.queryAll(By.css('p-tag'))
        expect(optionTags.length).toBe(4)

        // First option is configured to be always sent.
        expect(optionTags[0].properties.innerText).toBe('always sent')
        // One of the suboptions is empty.
        expect(optionTags[1].properties.innerText).toBe('empty suboption')
        // One of the top-level options is empty.
        expect(optionTags[2].properties.innerText).toBe('empty option')
        // Another empty suboption.
        expect(optionTags[3].properties.innerText).toBe('empty suboption')
    })

    it('should should display a message indicating there are no options', () => {
        let tree = fixture.debugElement.query(By.css('p-tree'))
        expect(tree).toBeTruthy()
        expect(tree.properties.innerText).toContain('No options configured.')
    })

    it('should should display DHCPv4 option name when it is known', () => {
        let options: DHCPOption[][] = [
            [
                {
                    code: 5,
                    fields: [
                        {
                            fieldType: 'ipv4-address',
                            values: ['192.0.2.1'],
                        },
                    ],
                    universe: 4,
                },
            ],
        ]
        component.options = options
        component.levels = ['subnet']
        component.ngOnInit()
        fixture.detectChanges()

        let optionSet = fixture.debugElement.query(By.css('p-tree'))
        expect(optionSet).toBeTruthy()
        expect(optionSet.properties.innerText).toContain('(5) Name Server')
    })
    it('should should display DHCPv6 option name when it is known', () => {
        let options: DHCPOption[][] = [
            [
                {
                    code: 23,
                    fields: [
                        {
                            fieldType: 'ipv6-address',
                            values: ['2001:db8:cafe::'],
                        },
                    ],
                    universe: 6,
                },
            ],
        ]
        component.options = options
        component.levels = ['subnet']
        component.ngOnInit()
        fixture.detectChanges()

        let optionSet = fixture.debugElement.query(By.css('p-tree'))
        expect(optionSet).toBeTruthy()
        expect(optionSet.properties.innerText).toContain('(23) OPTION_DNS_SERVERS')
    })

    it('should combine options from all levels', () => {
        let options: DHCPOption[][] = [
            [
                {
                    alwaysSend: true,
                    code: 1024,
                    fields: [
                        {
                            fieldType: 'uint32',
                            values: ['111'],
                        },
                        {
                            fieldType: 'ipv6-prefix',
                            values: ['3000::', '64'],
                        },
                    ],
                    universe: 6,
                },
                {
                    code: 1027,
                    fields: [
                        {
                            fieldType: 'bool',
                            values: ['true'],
                        },
                    ],
                    universe: 6,
                },
                {
                    code: 1028,
                    universe: 6,
                },
            ],
            [
                {
                    code: 1027,
                    fields: [
                        {
                            fieldType: 'bool',
                            values: ['false'],
                        },
                    ],
                    universe: 6,
                },
                {
                    code: 1030,
                    fields: [
                        {
                            fieldType: 'ipv4-address',
                            values: ['1.1.1.1'],
                        },
                    ],
                    universe: 6,
                },
            ],
        ]
        component.options = options
        component.levels = ['subnet', 'global']
        component.ngOnInit()
        fixture.detectChanges()

        // Initially, all options should be visible.
        expect(component.currentLevelOnlyMode).toBeFalse()
        expect(component.displayedOptionNodes.length).toBe(4)

        expect((component.displayedOptionNodes[0] as TreeNode<OptionNode>).data.code).toBe(1024)
        expect((component.displayedOptionNodes[1] as TreeNode<OptionNode>).data.code).toBe(1027)
        expect((component.displayedOptionNodes[2] as TreeNode<OptionNode>).data.code).toBe(1028)
        expect((component.displayedOptionNodes[3] as TreeNode<OptionNode>).data.code).toBe(1030)

        // Make sure that subnet-level option 1027 has been taken, rather than global.
        expect(component.displayedOptionNodes[1].children.length).toBe(1)
        expect(component.displayedOptionNodes[1].children[0].type).toBe('field')
        expect((component.displayedOptionNodes[1].children[0] as TreeNode<OptionFieldNode>).data.value).toBe('true')

        // Toggle displaying all options to subnet-level options only.
        component.currentLevelOnlyMode = true
        component.onCombinedChange()

        // Now, we should have 3 options only.
        expect(component.displayedOptionNodes.length).toBe(3)

        expect((component.displayedOptionNodes[0] as TreeNode<OptionNode>).data.code).toBe(1024)
        expect((component.displayedOptionNodes[1] as TreeNode<OptionNode>).data.code).toBe(1027)
        expect((component.displayedOptionNodes[2] as TreeNode<OptionNode>).data.code).toBe(1028)

        // Toggle again and we should be back to 4 options.
        component.currentLevelOnlyMode = false
        component.onCombinedChange()

        expect(component.displayedOptionNodes.length).toBe(4)
    })

    it('should not show the toggle button for a single inheritance level', () => {
        let options: DHCPOption[][] = [
            [
                {
                    code: 1028,
                    universe: 6,
                },
            ],
        ]
        component.options = options
        component.levels = ['host']
        component.ngOnInit()
        fixture.detectChanges()

        expect(fixture.debugElement.query(By.css('p-checkbox'))).toBeFalsy()
    })

    it('should show the toggle button for a single inheritance level', () => {
        let options: DHCPOption[][] = [
            [
                {
                    code: 1028,
                    universe: 6,
                },
            ],
            [
                {
                    code: 1028,
                    universe: 6,
                },
            ],
        ]
        component.options = options
        component.levels = ['subnet', 'global']
        component.ngOnInit()
        fixture.detectChanges()

        expect(fixture.debugElement.query(By.css('p-checkbox'))).toBeTruthy()
    })

    it('should return correct level tag severity', () => {
        component.levels = ['subnet', 'shared network', 'global']
        let node: TreeNode<OptionNode> = {
            type: 'option',
            data: {
                code: 1,
                universe: IPType.IPv4,
                level: 'subnet',
            },
        }
        expect(component.getLevelTagSeverity(node)).toBe('success')

        node.data.level = 'shared network'
        expect(component.getLevelTagSeverity(node)).toBe('warning')

        node.data.level = 'global'
        expect(component.getLevelTagSeverity(node)).toBe('danger')

        node.data.level = 'other'
        expect(component.getLevelTagSeverity(node)).toBe('info')
    })
})
