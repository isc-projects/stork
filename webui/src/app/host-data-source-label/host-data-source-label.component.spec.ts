import { ComponentFixture, TestBed } from '@angular/core/testing'

import { HostDataSourceLabelComponent } from './host-data-source-label.component'

describe('HostDataSourceLabelComponent', () => {
    let component: HostDataSourceLabelComponent
    let fixture: ComponentFixture<HostDataSourceLabelComponent>

    beforeEach(async () => {
        await TestBed.compileComponents()

        fixture = TestBed.createComponent(HostDataSourceLabelComponent)
        component = fixture.componentInstance
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display config label', () => {
        fixture.componentRef.setInput('dataSource', 'config')
        fixture.detectChanges()
        const compiled = fixture.nativeElement
        expect(compiled.querySelector('span').textContent).toContain('config')
    })

    it('should display api label', () => {
        fixture.componentRef.setInput('dataSource', 'api')
        fixture.detectChanges()
        const compiled = fixture.nativeElement
        expect(compiled.querySelector('span').textContent).toContain('host_cmds')
    })

    it('should display unknown label', () => {
        fixture.componentRef.setInput('dataSource', 'unknown')
        fixture.detectChanges()
        const compiled = fixture.nativeElement
        expect(compiled.innerText).toContain('unknown')
    })
})
