import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { ActivatedRoute, Router } from '@angular/router'

import { EventsPageComponent } from './events-page.component'
import { EventsService } from '../backend/api/events.service'
import { EventsPanelComponent } from '../events-panel/events-panel.component'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { FormsModule } from '@angular/forms'
import { SelectButtonModule } from 'primeng/selectbutton'
import { DropdownModule } from 'primeng/dropdown'
import { TableModule } from 'primeng/table'

describe('EventsPageComponent', () => {
    let component: EventsPageComponent
    let fixture: ComponentFixture<EventsPageComponent>

    beforeEach(
        waitForAsync(() => {
            TestBed.configureTestingModule({
                providers: [
                    EventsService,
                    {
                        provide: ActivatedRoute,
                        useValue: {
                            snapshot: { queryParams: {} },
                        },
                    },
                    {
                        provide: Router,
                        useValue: {
                            navigate: () => {},
                        },
                    },
                    MessageService,
                ],
                declarations: [EventsPageComponent, EventsPageComponent, EventsPanelComponent],
                imports: [HttpClientTestingModule, FormsModule, SelectButtonModule, DropdownModule, TableModule],
            }).compileComponents()
        })
    )

    beforeEach(() => {
        fixture = TestBed.createComponent(EventsPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
