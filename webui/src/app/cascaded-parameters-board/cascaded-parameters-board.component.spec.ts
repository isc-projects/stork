import { ComponentFixture, TestBed } from '@angular/core/testing'

import { CascadedParametersBoardComponent } from './cascaded-parameters-board.component'
import { KeaConfigSubnetDerivedParameters } from '../backend'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { ButtonModule } from 'primeng/button'
import { TableModule } from 'primeng/table'
import { By } from '@angular/platform-browser'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { TooltipModule } from 'primeng/tooltip'
import { UncamelPipe } from '../pipes/uncamel.pipe'
import { UnhyphenPipe } from '../pipes/unhyphen.pipe'
import { ParameterViewComponent } from '../parameter-view/parameter-view.component'

describe('CascadedParametersBoardComponent', () => {
    let component: CascadedParametersBoardComponent<KeaConfigSubnetDerivedParameters>
    let fixture: ComponentFixture<CascadedParametersBoardComponent<KeaConfigSubnetDerivedParameters>>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [
                CascadedParametersBoardComponent,
                ParameterViewComponent,
                PlaceholderPipe,
                UncamelPipe,
                UnhyphenPipe,
            ],
            imports: [ButtonModule, NoopAnimationsModule, TableModule, TooltipModule],
        }).compileComponents()

        fixture = TestBed.createComponent(CascadedParametersBoardComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should initialize the board for display', () => {
        component.levels = ['Subnet', 'Shared Network', 'Global']
        component.data = [
            {
                name: 'Server1',
                parameters: [
                    {
                        cacheThreshold: 0.25,
                        cacheMaxAge: 1000,
                        clientClass: 'baz',
                        requireClientClasses: ['foo', 'bar'],
                        ddnsOverrideClientUpdate: true,
                        relay: { ipAddresses: ['192.0.2.1'] },
                    },
                    {
                        cacheThreshold: 0.25,
                        cacheMaxAge: 1000,
                        clientClass: 'fbi',
                        requireClientClasses: ['abc'],
                        ddnsOverrideClientUpdate: false,
                    },
                    {
                        cacheMaxAge: 1000,
                        requireClientClasses: ['abc'],
                        ddnsOverrideClientUpdate: true,
                        relay: { ipAddresses: ['192.0.2.2'] },
                    },
                ],
            },
            {
                name: 'Server2',
                parameters: [
                    {
                        cacheThreshold: 0.22,
                        cacheMaxAge: 900,
                        clientClass: 'abc',
                        requireClientClasses: ['bar'],
                        ddnsOverrideClientUpdate: true,
                    },
                    {
                        cacheThreshold: 0.21,
                        cacheMaxAge: 800,
                        clientClass: 'ibi',
                        requireClientClasses: ['abc', 'dec'],
                        ddnsOverrideClientUpdate: true,
                    },
                    {
                        cacheMaxAge: 1000,
                        requireClientClasses: ['aaa'],
                        ddnsOverrideClientUpdate: false,
                    },
                ],
            },
        ]
        component.ngOnInit()

        expect(component.rows.length).toBe(6)
        expect(component.rows[0].name).toBe('Cache Max Age')
        expect(component.rows[0].parameters.length).toBe(2)
        expect(component.rows[0].parameters[0].effective).toBe(1000)
        expect(component.rows[0].parameters[0].level).toBe('Subnet')
        expect(component.rows[0].parameters[0].values.length).toBe(3)
        expect(component.rows[0].parameters[0].values[0]).toBe(1000)
        expect(component.rows[0].parameters[0].values[1]).toBe(1000)
        expect(component.rows[0].parameters[0].values[2]).toBe(1000)

        expect(component.rows[0].parameters[1].effective).toBe(900)
        expect(component.rows[0].parameters[1].level).toBe('Subnet')
        expect(component.rows[0].parameters[1].values.length).toBe(3)
        expect(component.rows[0].parameters[1].values[0]).toBe(900)
        expect(component.rows[0].parameters[1].values[1]).toBe(800)
        expect(component.rows[0].parameters[1].values[2]).toBe(1000)

        expect(component.rows[1].name).toBe('Cache Threshold')
        expect(component.rows[1].parameters.length).toBe(2)
        expect(component.rows[1].parameters[0].effective).toBe(0.25)
        expect(component.rows[1].parameters[0].level).toBe('Subnet')
        expect(component.rows[1].parameters[0].values.length).toBe(3)
        expect(component.rows[1].parameters[0].values[0]).toBe(0.25)
        expect(component.rows[1].parameters[0].values[1]).toBe(0.25)
        expect(component.rows[1].parameters[0].values[2]).toBeNull()
        expect(component.rows[1].parameters[1].effective).toBe(0.22)
        expect(component.rows[1].parameters[1].level).toBe('Subnet')
        expect(component.rows[1].parameters[1].values.length).toBe(3)
        expect(component.rows[1].parameters[1].values[0]).toBe(0.22)
        expect(component.rows[1].parameters[1].values[1]).toBe(0.21)
        expect(component.rows[1].parameters[1].values[2]).toBeNull()

        expect(component.rows[2].name).toBe('Client Class')
        expect(component.rows[2].parameters.length).toBe(2)
        expect(component.rows[2].parameters[0].effective).toBe('baz')
        expect(component.rows[2].parameters[0].level).toBe('Subnet')
        expect(component.rows[2].parameters[0].values.length).toBe(3)
        expect(component.rows[2].parameters[0].values[0]).toBe('baz')
        expect(component.rows[2].parameters[0].values[1]).toBe('fbi')
        expect(component.rows[2].parameters[0].values[2]).toBeNull()
        expect(component.rows[2].parameters[1].effective).toBe('abc')
        expect(component.rows[2].parameters[1].level).toBe('Subnet')
        expect(component.rows[2].parameters[1].values.length).toBe(3)
        expect(component.rows[2].parameters[1].values[0]).toBe('abc')
        expect(component.rows[2].parameters[1].values[1]).toBe('ibi')
        expect(component.rows[2].parameters[1].values[2]).toBeNull()

        expect(component.rows[3].name).toBe('DDNS Override Client Update')
        expect(component.rows[3].parameters.length).toBe(2)
        expect(component.rows[3].parameters[0].effective).toBeTrue()
        expect(component.rows[3].parameters[0].level).toBe('Subnet')
        expect(component.rows[3].parameters[0].values.length).toBe(3)
        expect(component.rows[3].parameters[0].values[0]).toBeTrue()
        expect(component.rows[3].parameters[0].values[1]).toBeFalse()
        expect(component.rows[3].parameters[0].values[2]).toBeTrue()
        expect(component.rows[3].parameters[1].effective).toBeTrue()
        expect(component.rows[3].parameters[1].level).toBe('Subnet')
        expect(component.rows[3].parameters[1].values.length).toBe(3)
        expect(component.rows[3].parameters[1].values[0]).toBeTrue()
        expect(component.rows[3].parameters[1].values[1]).toBeTrue()
        expect(component.rows[3].parameters[1].values[2]).toBeFalse()

        expect(component.rows[4].name).toBe('Relay')
        expect(component.rows[4].parameters.length).toBe(2)
        expect(component.rows[4].parameters[0].effective).toContain('IP Addresses')
        expect(component.rows[4].parameters[0].effective).toContain('192.0.2.1')
        expect(component.rows[4].parameters[0].level).toBe('Subnet')
        expect(component.rows[4].parameters[0].values.length).toBe(3)
        expect(component.rows[4].parameters[0].values[0]).toContain('IP Addresses')
        expect(component.rows[4].parameters[0].values[0]).toContain('192.0.2.1')
        expect(component.rows[4].parameters[0].values[1]).toBeNull()
        expect(component.rows[4].parameters[0].values[2]).toContain('192.0.2.2')
        expect(component.rows[4].parameters[1].effective).toBeNull()
        expect(component.rows[4].parameters[1].level).toBeNull()
        expect(component.rows[4].parameters[1].values.length).toBe(3)
        expect(component.rows[4].parameters[1].values[0]).toBeNull()
        expect(component.rows[4].parameters[1].values[1]).toBeNull()
        expect(component.rows[4].parameters[1].values[2]).toBeNull()

        expect(component.rows[5].name).toBe('Require Client Classes')
        expect(component.rows[5].parameters.length).toBe(2)
        expect(component.rows[5].parameters[0].effective).toBe('[ foo, bar ]')
        expect(component.rows[5].parameters[0].level).toBe('Subnet')
        expect(component.rows[5].parameters[0].values.length).toBe(3)
        expect(component.rows[5].parameters[0].values[0]).toBe('[ foo, bar ]')
        expect(component.rows[5].parameters[0].values[1]).toBe('[ abc ]')
        expect(component.rows[5].parameters[0].values[2]).toBe('[ abc ]')
        expect(component.rows[5].parameters[1].effective).toBe('[ bar ]')
        expect(component.rows[5].parameters[1].level).toBe('Subnet')
        expect(component.rows[5].parameters[1].values.length).toBe(3)
        expect(component.rows[5].parameters[1].values[0]).toBe('[ bar ]')
        expect(component.rows[5].parameters[1].values[1]).toBe('[ abc, dec ]')
        expect(component.rows[5].parameters[1].values[2]).toBe('[ aaa ]')
    })

    it('should exclude selected parameters', () => {
        component.levels = ['Subnet', 'Global']
        component.data = [
            {
                name: 'Server1',
                parameters: [
                    {
                        cacheThreshold: 0.25,
                        cacheMaxAge: 1000,
                        clientClass: 'baz',
                    },
                    {
                        cacheThreshold: null,
                        cacheMaxAge: 1000,
                        clientClass: null,
                    },
                ],
            },
        ]
        component.excludedParameters = ['clientClass', 'cacheThreshold']

        component.ngOnInit()
        fixture.detectChanges()

        let rows = fixture.debugElement.queryAll(By.css('tr'))
        expect(rows.length).toBe(2)

        expect(rows[1].nativeElement.innerText).toContain('Cache Max Age')
    })

    it('should expand the parameters with a button', () => {
        component.levels = ['Subnet', 'Global']
        component.data = [
            {
                name: 'Server1',
                parameters: [
                    {
                        validLifetime: 1000,
                    },
                    {
                        validLifetime: 900,
                    },
                ],
            },
        ]
        component.ngOnInit()
        fixture.detectChanges()

        let rows = fixture.debugElement.queryAll(By.css('tr'))
        expect(rows.length).toBe(2)

        let columns = rows[1].queryAll(By.css('td'))
        expect(columns.length).toBe(3)
        expect(columns[1].nativeElement.innerText).toBe('Valid Lifetime')
        expect(columns[2].nativeElement.innerText).toBe('1000')

        const button = columns[0].query(By.css('button'))
        expect(button).toBeTruthy()

        button.nativeElement.click()
        fixture.detectChanges()

        rows = fixture.debugElement.queryAll(By.css('tr'))
        expect(rows.length).toBe(4)

        columns = rows[2].queryAll(By.css('td'))
        expect(columns.length).toBe(3)
        expect(columns[1].nativeElement.innerText).toBe('Subnet')
        expect(columns[2].nativeElement.innerText).toBe('1000')

        columns = rows[3].queryAll(By.css('td'))
        expect(columns[1].nativeElement.innerText).toBe('Global')
        expect(columns[2].nativeElement.innerText).toBe('900')
    })

    it('should not expand the parameters when only one level is specified', () => {
        component.levels = ['Global']
        component.data = [
            {
                name: 'Server1',
                parameters: [
                    {
                        validLifetime: 1000,
                    },
                    {
                        validLifetime: 900,
                    },
                ],
            },
        ]
        component.ngOnInit()
        fixture.detectChanges()

        let rows = fixture.debugElement.queryAll(By.css('tr'))
        expect(rows.length).toBe(2)

        let columns = rows[1].queryAll(By.css('td'))
        expect(columns.length).toBe(2)
        expect(columns[0].nativeElement.innerText).toBe('Valid Lifetime')
        expect(columns[1].nativeElement.innerText).toBe('1000')
    })

    it('should display the data set names', () => {
        component.levels = ['Subnet', 'Global']
        component.data = [
            {
                name: 'Server1',
                parameters: [
                    {
                        validLifetime: 1000,
                    },
                ],
            },
            {
                name: 'Server2',
                parameters: [
                    {
                        validLifetime: 2000,
                    },
                ],
            },
        ]
        component.ngOnInit()
        fixture.detectChanges()

        const headers = fixture.debugElement.queryAll(By.css('th'))
        expect(headers.length).toBe(4)

        expect(headers[2].nativeElement.innerText).toBe('Server1')
        expect(headers[3].nativeElement.innerText).toBe('Server2')
    })

    it('should display the text about no parameters', () => {
        component.levels = ['Subnet', 'Global']
        component.data = [
            {
                name: 'Server1',
                parameters: [],
            },
        ]
        component.ngOnInit()
        fixture.detectChanges()

        const span = fixture.debugElement.query(By.css('span'))
        expect(span).toBeTruthy()
        expect(span.nativeElement.innerText).toContain('No parameters configured.')
    })
})
