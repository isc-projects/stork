import { ComponentFixture, TestBed } from '@angular/core/testing'

import { TabViewComponent } from './tab-view.component'
import { RouterModule } from '@angular/router'
import { MessageService } from 'primeng/api'

describe('TabViewComponent', () => {
    let component: TabViewComponent<any, any>
    let fixture: ComponentFixture<TabViewComponent<any, any>>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [TabViewComponent, RouterModule.forRoot([])],
            providers: [MessageService],
        }).compileComponents()

        fixture = TestBed.createComponent(TabViewComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
