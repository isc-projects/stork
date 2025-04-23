import { StorkTemplateDirective } from './stork-template.directive'
import { TemplateRef } from '@angular/core'

describe('StorkTemplateDirective', () => {
    it('should create an instance', () => {
        const directive = new StorkTemplateDirective(null as TemplateRef<any>)
        expect(directive).toBeTruthy()
    })
})
