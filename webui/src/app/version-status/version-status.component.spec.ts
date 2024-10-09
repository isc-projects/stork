import { ComponentFixture, TestBed } from '@angular/core/testing'

import { VersionStatusComponent } from './version-status.component'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { TooltipModule } from 'primeng/tooltip'

describe('VersionStatusComponent', () => {
    let component: VersionStatusComponent
    let fixture: ComponentFixture<VersionStatusComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [HttpClientTestingModule, TooltipModule],
            declarations: [VersionStatusComponent],
            providers: [MessageService],
        }).compileComponents()

        fixture = TestBed.createComponent(VersionStatusComponent)
        component = fixture.componentInstance
        fixture.componentRef.setInput('app', 'kea')
        fixture.componentRef.setInput('version', '2.6.1')
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
