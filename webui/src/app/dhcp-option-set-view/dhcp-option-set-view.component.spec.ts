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

describe('DhcpOptionSetViewComponent', () => {
    let component: DhcpOptionSetViewComponent
    let fixture: ComponentFixture<DhcpOptionSetViewComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [OverlayPanelModule, TagModule, TooltipModule, TreeModule],
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
        let options: Array<DHCPOption> = [
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
                options: [
                    {
                        code: 1025,
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
                    },
                ],
            },
        ]
        component.options = options
        component.ngOnInit()
        fixture.detectChanges()

        expect(component.optionNodes.length).toBe(3)

        // Option 1024.
        expect((component.optionNodes[0] as TreeNode<OptionNode>).data.alwaysSend).toBeTrue()
        expect((component.optionNodes[0] as TreeNode<OptionNode>).data.code).toBe(1024)

        // Option 1024 fields.
        expect(component.optionNodes[0].children.length).toBe(4)
        expect(component.optionNodes[0].children[0].type).toBe('field')
        expect(component.optionNodes[0].children[0].expanded).toBeTrue()
        expect((component.optionNodes[0].children[0] as TreeNode<OptionFieldNode>).data.fieldType).toBe('uint32')
        expect((component.optionNodes[0].children[0] as TreeNode<OptionFieldNode>).data.value).toBe('111')
        expect(component.optionNodes[0].children[1].type).toBe('field')
        expect(component.optionNodes[0].children[1].expanded).toBeTrue()
        expect((component.optionNodes[0].children[1] as TreeNode<OptionFieldNode>).data.fieldType).toBe('ipv6-prefix')
        expect((component.optionNodes[0].children[1] as TreeNode<OptionFieldNode>).data.value).toBe('3000::/64')

        // Option 1024 suboptions.
        expect((component.optionNodes[0].children[2] as TreeNode<OptionNode>).data.code).toBe(1025)
        expect(component.optionNodes[0].children[2].children.length).toBe(0)

        // Suboption 1026.
        expect((component.optionNodes[0].children[3] as TreeNode<OptionNode>).data.code).toBe(1026)

        // Suboption 1026 fields.
        expect(component.optionNodes[0].children[3].children.length).toBe(2)
        expect((component.optionNodes[0].children[3].children[0] as TreeNode<OptionFieldNode>).data.fieldType).toBe(
            'ipv6-address'
        )
        expect((component.optionNodes[0].children[3].children[0] as TreeNode<OptionFieldNode>).data.value).toBe(
            '2001:db8:1::1'
        )
        expect((component.optionNodes[0].children[3].children[1] as TreeNode<OptionFieldNode>).data.fieldType).toBe(
            'ipv6-address'
        )
        expect((component.optionNodes[0].children[3].children[1] as TreeNode<OptionFieldNode>).data.value).toBe(
            '2001:db8:2::1'
        )
        // Option 1027.
        expect((component.optionNodes[1] as TreeNode<OptionNode>).data.alwaysSend).toBeFalsy()
        expect((component.optionNodes[1] as TreeNode<OptionNode>).data.code).toBe(1027)
        expect(component.optionNodes[1].children.length).toBe(1)
        expect(component.optionNodes[1].children[0].type).toBe('field')
        expect(component.optionNodes[1].children[0].expanded).toBeTrue()
        expect((component.optionNodes[1].children[0] as TreeNode<OptionFieldNode>).data.fieldType).toBe('bool')
        expect((component.optionNodes[1].children[0] as TreeNode<OptionFieldNode>).data.value).toBe('true')

        // Option 1028.
        expect((component.optionNodes[2] as TreeNode<OptionNode>).data.alwaysSend).toBeFalsy()
        expect((component.optionNodes[2] as TreeNode<OptionNode>).data.code).toBe(1028)
        expect(component.optionNodes[2].children.length).toBe(1)

        // Suboption 1029.
        expect((component.optionNodes[2].children[0] as TreeNode<OptionNode>).data.code).toBe(1029)

        // Make sure that appropriate tags are displayed.
        let optionTags = fixture.debugElement.queryAll(By.css('p-tag'))
        expect(optionTags.length).toBe(3)

        // First option is configured to be always sent.
        expect(optionTags[0].properties.innerText).toBe('always sent')
        // One of the suboptions is empty.
        expect(optionTags[1].properties.innerText).toBe('empty suboption')
        // One of the top-level options is empty.
        expect(optionTags[2].properties.innerText).toBe('empty option')
    })

    it('should should display a message indicating there are no options', () => {
        let tree = fixture.debugElement.query(By.css('p-tree'))
        expect(tree).toBeTruthy()
        expect(tree.properties.innerText).toContain('No options configured.')
    })
})
