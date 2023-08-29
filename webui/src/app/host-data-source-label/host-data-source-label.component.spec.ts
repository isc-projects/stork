import { ComponentFixture, TestBed } from '@angular/core/testing'

import { HostDataSourceLabelComponent } from './host-data-source-label.component'
import { TagModule } from 'primeng/tag'

describe('HostDataSourceLabelComponent', () => {
    let component: HostDataSourceLabelComponent
    let fixture: ComponentFixture<HostDataSourceLabelComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [HostDataSourceLabelComponent],
            imports: [TagModule],
        }).compileComponents()

        fixture = TestBed.createComponent(HostDataSourceLabelComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display config label', () => {
        component.dataSource = 'config'
        fixture.detectChanges()
        const compiled = fixture.nativeElement
        expect(compiled.querySelector('span').textContent).toContain('config')
    })

    it('should display api label', () => {
        component.dataSource = 'api'
        fixture.detectChanges()
        const compiled = fixture.nativeElement
        expect(compiled.querySelector('span').textContent).toContain('host_cmds')
    })

    it('should display unknown label', () => {
        component.dataSource = 'unknown'
        fixture.detectChanges()
        const compiled = fixture.nativeElement
        expect(compiled.querySelector('span').textContent).toContain('unknown')
    })
})
