import { ComponentFixture, TestBed } from '@angular/core/testing'

import { ByteCharacterComponent } from './byte-character.component'

describe('ByteCharacterComponent', () => {
    let component: ByteCharacterComponent
    let fixture: ComponentFixture<ByteCharacterComponent>

    beforeEach(async () => {
        await TestBed.compileComponents()

        fixture = TestBed.createComponent(ByteCharacterComponent)
        component = fixture.componentInstance
        component.byteValue = 42
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display printable character', () => {
        component.byteValue = 'A'.charCodeAt(0)
        fixture.detectChanges()

        expect(component.isPrintable).toBe(true)
        expect(component.character).toBe('A')
        expect(component.hex).toBe('41')
        expect(component.isNaN).toBe(false)

        const element = (fixture.nativeElement as HTMLElement).textContent
        expect(element).toBe('A')
    })

    it('should display printable symbol', () => {
        component.byteValue = 64
        fixture.detectChanges()

        expect(component.isPrintable).toBe(true)
        expect(component.character).toBe('@')
        expect(component.hex).toBe('40')
        expect(component.isNaN).toBe(false)

        const element = (fixture.nativeElement as HTMLElement).textContent
        expect(element).toBe('@')
    })

    it('should display non-printable character', () => {
        component.byteValue = 0
        fixture.detectChanges()

        expect(component.isPrintable).toBe(false)
        expect(component.character).toBe('\0')
        expect(component.hex).toBe('00')
        expect(component.isNaN).toBe(false)

        const element = (fixture.nativeElement as HTMLElement).textContent
        expect(element).toBe('\\0x00')
    })

    it('should display non-byte negative', () => {
        component.byteValue = -1
        fixture.detectChanges()

        expect(component.isPrintable).toBe(false)
        expect(component.character).toBe('￿')
        expect(component.hex).toBe('-1')
        expect(component.isNaN).toBe(false)

        // Trash in, trash out.
        const element = (fixture.nativeElement as HTMLElement).textContent
        expect(element).toBe('\\0x-1')
    })

    it('should display NaN', () => {
        component.byteValue = Number.NaN
        fixture.detectChanges()

        expect(component.isPrintable).toBe(false)
        expect(component.character).toBe('\0')
        expect(component.hex).toBe('NaN')
        expect(component.isNaN).toBe(true)

        const element = (fixture.nativeElement as HTMLElement).textContent
        expect(element).toBe('\\0xNaN')
    })

    it('should display non-byte positive', () => {
        component.byteValue = 256
        fixture.detectChanges()

        expect(component.isPrintable).toBe(false)
        expect(component.character).toBe('Ā')
        expect(component.hex).toBe('100')
        expect(component.isNaN).toBe(false)

        // Trash in, trash out.
        const element = (fixture.nativeElement as HTMLElement).textContent
        expect(element).toBe('\\0x100')
    })
})
