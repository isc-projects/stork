import { ComponentFixture, TestBed } from '@angular/core/testing'

import { TextFileViewerComponent } from './text-file-viewer.component'

describe('TextFileViewerComponent', () => {
    let component: TextFileViewerComponent
    let fixture: ComponentFixture<TextFileViewerComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [TextFileViewerComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(TextFileViewerComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should trim empty lines from the contents', () => {
        component.contents = ['', '\n', 'line 1', '', '', 'line 2', '\n', '\t', '\r', '']
        expect(component.contents).toEqual(['line 1', '', '', 'line 2'])
    })
})
