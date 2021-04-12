import { async, ComponentFixture, TestBed } from '@angular/core/testing'
import { By } from '@angular/platform-browser'
import { PaginatorModule } from 'primeng/paginator'

import { JsonTreeComponent } from './json-tree.component'

describe('JsonTreeComponent', () => {
    let component: JsonTreeComponent
    let fixture: ComponentFixture<JsonTreeComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            imports: [PaginatorModule],
            declarations: [JsonTreeComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(JsonTreeComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display spinner when object is not set', async () => {
        expect(component.hasAssignedValue()).toBeFalse()

        await fixture.whenStable()

        const spinner = fixture.debugElement.query(By.css('.tree-level__value .fa-spinner'))
        expect(spinner).not.toBeNull()
    })

    it('should primitive value be leaf', () => {
        component.value = 5
        fixture.detectChanges()
        const root = fixture.debugElement.query(By.css('.tree-level'))
        expect(root.classes['tree-level--leaf']).toBeTrue()
    })

    it('should empty object be leaf', () => {
        component.value = {}
        fixture.detectChanges()
        const root = fixture.debugElement.query(By.css('.tree-level'))
        expect(root.classes['tree-level--leaf']).toBeTrue()
    })

    it('should empty list be leaf', () => {
        component.value = []
        fixture.detectChanges()
        const root = fixture.debugElement.query(By.css('.tree-level'))
        expect(root.classes['tree-level--leaf']).toBeTrue()
    })

    it('should top level omit key', () => {
        component.value = 42
        fixture.detectChanges()
        const root = fixture.debugElement.query(By.css('.tree-level'))
        const key = root.query(By.css('.tree-level__key'))
        expect(key).toBeNull()
    })

    it('should display string key', () => {
        component.value = 42
        component.key = 'foo'
        fixture.detectChanges()
        const root = fixture.debugElement.query(By.css('.tree-level'))
        const key = root.query(By.css('.tree-level__key'))
        const keyElement = key.nativeElement as HTMLElement
        expect(key).not.toBeNull()
        expect(keyElement.textContent).toBe('foo')
        // CSS style adds a colon after key, but as pseudo-element
    })

    it('should display numeric key', () => {
        component.value = 'foo'
        component.key = '42'
        fixture.detectChanges()
        const root = fixture.debugElement.query(By.css('.tree-level'))
        const key = root.query(By.css('.tree-level__key'))
        const keyElement = key.nativeElement as HTMLElement
        expect(key).not.toBeNull()
        expect(keyElement.textContent).toBe('42')
    })

    it('should display special character key', () => {
        const rawKey = '!@#$%^&*()_+-=[];\',./\\{}|:"<>?`~'
        component.value = 'foo'
        component.key = rawKey
        fixture.detectChanges()
        const root = fixture.debugElement.query(By.css('.tree-level'))
        const key = root.query(By.css('.tree-level__key'))
        const keyElement = key.nativeElement as HTMLElement
        expect(key).not.toBeNull()
        expect(keyElement.textContent).toBe(rawKey)
    })

    it('should expand single child node', () => {
        component.value = [{ foo: 42 }]
        component.key = 'baz'
        component.autoExpandMaxNodeCount = 0
        fixture.detectChanges()

        const root = fixture.debugElement.query(By.css('.tree-level'))
        const rootElement = root.nativeElement as HTMLDetailsElement
        expect(rootElement.open).toBeFalse()

        const nested = root.query(By.css('.tree-level'))
        const nestedElement = nested.nativeElement as HTMLDetailsElement
        expect(nestedElement.open).toBeTrue()
    })

    it('should expand children nodes when children count not exceed limit', () => {
        component.value = ['foo', 'bar', 'foobar']
        component.key = 'baz'
        component.autoExpandMaxNodeCount = 5
        component.forceOpenThisLevel = false
        fixture.detectChanges()

        const root = fixture.debugElement.query(By.css('.tree-level'))
        const rootElement = root.nativeElement as HTMLDetailsElement
        expect(rootElement.open).toBeTrue()
    })

    it('should not expand children nodes when children count exceed limit', () => {
        component.value = ['foo', 'bar', 'foobar']
        component.key = 'baz'
        component.autoExpandMaxNodeCount = 2
        component.forceOpenThisLevel = false
        fixture.detectChanges()

        const root = fixture.debugElement.query(By.css('.tree-level'))
        const rootElement = root.nativeElement as HTMLDetailsElement
        expect(rootElement.open).toBeFalse()
    })

    it('should expand this level node property working', () => {
        component.value = ['foo', 'bar', 'foobar']
        component.key = 'baz'
        component.autoExpandMaxNodeCount = 0
        component.forceOpenThisLevel = true
        fixture.detectChanges()

        const root = fixture.debugElement.query(By.css('.tree-level'))
        const rootElement = root.nativeElement as HTMLDetailsElement
        expect(rootElement.open).toBeTrue()
    })

    it('should increase recursion level when nesting', () => {
        component.value = {
            'level-1': {
                'level-2': {
                    'level-3': 42,
                },
            },
        }
        component.key = 'baz'
        fixture.detectChanges()

        let level = fixture.debugElement.query(By.css('.tree-level'))

        for (let index = 1; index <= 3; index++) {
            level = level.query(By.css('.tree-level'))
            const instance = level.componentInstance as JsonTreeComponent
            expect(instance.recursionLevel).toBe(index)
        }
    })

    it('should stop render nodes when recursion level exceed', async () => {
        // Generate object
        const recursionLimit = 50

        const content = {}
        let contentLevel = content
        for (let index = 1; index < 2 * recursionLimit; index++) {
            const key = `level-${index}`
            contentLevel[key] = {}
            contentLevel = contentLevel[key]
        }

        // Leaf is not primitive
        contentLevel['foo'] = 42

        component.key = 'baz'
        component.value = content
        fixture.detectChanges()
        let instance: JsonTreeComponent = null

        // Iterate over nested nodes
        let level = fixture.debugElement.query(By.css('.tree-level'))
        for (let index = 1; index < recursionLimit; index++) {
            level = level.query(By.css('.tree-level'))
            instance = level.componentInstance as JsonTreeComponent
            expect(instance.recursionLevel).toBe(index)
        }

        const lastStandardNode = level

        // Reach limit - last node
        level = level.query(By.css('.tree-level'))
        instance = level.componentInstance as JsonTreeComponent
        expect(level).not.toBeNull()
        expect((level.nativeElement as HTMLElement).tagName).toBe('DIV')
        expect(instance.isRecursionLevelReached()).toBeTrue()

        // Check if value mark as load more link
        const value = level.query(By.css('.tree-level__value--clickable'))
        expect(value).not.toBeNull()
        expect((value.nativeElement as HTMLElement).textContent).toContain('Load more')

        // Next node level doesn't exist
        expect(level.query(By.css('.tree-level'))).toBeNull()

        // Clickable link
        const clickableSpan = value.query(By.css('span'))
        const clickableSpanElement = clickableSpan.nativeElement as HTMLElement
        expect(clickableSpan).not.toBeNull()
        expect(clickableSpanElement.tagName).toBe('SPAN')

        // Trig load handler
        const resetHandlerSpy = spyOn(instance, 'onClickResetRecursionLevel').and.callThrough()
        clickableSpanElement.click()
        await fixture.whenStable()
        await expect(resetHandlerSpy).toHaveBeenCalled()

        // Check if next nodes loaded
        fixture.detectChanges()
        await fixture.whenRenderingDone()
        expect(instance.recursionLevel).toBe(0)

        const levels = fixture.debugElement.queryAll(By.css('.tree-level'))
        expect(levels.length).toBe(2 * recursionLimit + 1)
    })

    it('should reset recursion level after click', () => {
        component.recursionLevel = 10
        expect(component.recursionLevel).toBe(10)
        component.onClickResetRecursionLevel()
        expect(component.recursionLevel).toBe(0)
    })

    it('should not display pagination when children count is low', () => {
        const items = [1]
        component.value = items
        expect(component.hasPaginateChildren()).toBeFalse()

        fixture.detectChanges()
        const paginator = fixture.debugElement.query(By.css('.ui-paginator'))
        expect(paginator).toBeNull()
    })

    it('should display pagination when children count is high', () => {
        const items = new Array(component.childStep * 3).fill(0)
        component.value = items
        expect(component.hasPaginateChildren()).toBeTrue()

        fixture.detectChanges()
        const paginator = fixture.debugElement.query(By.css('.ui-paginator'))
        expect(paginator).not.toBeNull()
    })

    it('should start pagination from first child', () => {
        const items = new Array(42).fill(0)
        component.value = items
        expect(component.childStart).toBe(0)
    })

    it('should pagination page contains correct number of children', () => {
        const count = 424
        const items = new Array(count).fill(0)
        component.value = items
        expect(component.totalChildrenCount).toBe(count)
    })

    it('should change page after click on number of page', async () => {
        const count = 424
        const items = new Array(count).fill(0)
        component.value = items

        fixture.detectChanges()
        const paginator = fixture.debugElement.query(By.css('.ui-paginator'))
        const pages = paginator.queryAll(By.css('.ui-paginator-page'))
        expect(pages.length).toBeGreaterThan(1)
        const page = pages[pages.length - 1]
        const pageElement = page.nativeElement as HTMLElement

        const pageChangedHandler = spyOn(component, 'onPageChildrenChanged').and.callThrough()
        pageElement.click()
        await fixture.whenStable()
        expect(pageChangedHandler).toHaveBeenCalled()
    })

    it('should change page after input number of page and press Enter', async () => {
        const count = 424
        const items = new Array(count).fill(0)
        component.value = items

        fixture.detectChanges()

        const pressEnterHandler = spyOn(component, 'onEnterJumpToPage').and.callThrough()
        const jumpInput = fixture.debugElement.query(By.css('.ui-paginator__jump-to-page'))
        const jumpInputElement = jumpInput.nativeElement as HTMLInputElement
        expect(jumpInputElement.tagName).toBe('INPUT')
        // Human input - index starts from 1
        jumpInputElement.value = '2'
        jumpInputElement.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }))
        jumpInputElement.dispatchEvent(new KeyboardEvent('keyup', { key: 'Enter' }))
        await fixture.whenStable()
        expect(pressEnterHandler).toHaveBeenCalled()
    })

    it('should correctly display last page of pagination', async () => {
        const step = component.childStep
        const count = step * 2 + Math.floor(step / 2)
        const items = new Array(count).fill(0)
        component.value = items
        component.onEnterJumpToPage(2)
        fixture.detectChanges()
        await fixture.whenRenderingDone()

        expect(component.childStep).toBe(step)
        expect(component.childStart).toBe(step * 2)
        expect(component.childEnd).toBe(count)
    })

    it('should order object (dictionary) by key', () => {
        component.value = { c: 3, b: 2, a: 1 }
        component.key = null
        fixture.detectChanges()
        // I hope that it keeps order
        const keys = fixture.debugElement.queryAll(By.css('.tree-level__key'))
        expect(keys.length).toBe(3)
        const rawKeys = keys.map((k) => (k.nativeElement as HTMLElement).textContent)
        expect(rawKeys[0]).toBe('a')
        expect(rawKeys[1]).toBe('b')
        expect(rawKeys[2]).toBe('c')
    })

    it('should order array by index', () => {
        const count = 50
        component.value = new Array(count).fill(0)
        component.key = null
        fixture.detectChanges()
        // I hope that it keeps order
        const keys = fixture.debugElement.queryAll(By.css('.tree-level__key'))
        expect(keys.length).toBe(count)
        const rawKeys = keys.map((k) => (k.nativeElement as HTMLElement).textContent)

        for (let index = 0; index < count; index++) {
            expect(rawKeys[index]).toBe(index + '')
        }
    })

    it('should count children', () => {
        const items = new Array(2137).fill(0)
        component.value = items
        expect(component.totalChildrenCount).toBe(2137)
        component.value = { foo: 1, bar: 2, baz: 3 }
        expect(component.totalChildrenCount).toBe(3)
        component.value = []
        expect(component.totalChildrenCount).toBe(0)
        component.value = 42
        expect(component.totalChildrenCount).toBe(0)
    })

    it('should indicate loading when page is changed', async () => {
        const items = new Array(100).fill(0)
        component.value = items
        fixture.detectChanges()

        await fixture.whenRenderingDone()
        expect(component.areChildrenLoading()).toBeFalse()

        const spyRenderHandler = spyOn(component, 'onFinishRenderChildren').and.callThrough()

        component.onEnterJumpToPage(1)
        expect(component.areChildrenLoading()).toBeTrue()
        fixture.detectChanges()

        await fixture.whenRenderingDone()
        // First time - when loading indicator starts
        // Second time - when new items are rendered
        expect(spyRenderHandler).toHaveBeenCalledTimes(2)
        expect(component.areChildrenLoading()).toBe(false)
    })

    it('should recognize primitive types', () => {
        const primitiveTypes = [
            1, // Number
            0.5, // Number float
            true, // boolean
            null, // Nullable
            new Date(), // Date
        ]

        for (const item of primitiveTypes) {
            component.value = item
            expect(component.isPrimitive()).toBeTrue()

            fixture.detectChanges()
            const root = fixture.debugElement.query(By.css('.tree-level--leaf'))
            expect(root).not.toBeNull()
        }
    })

    it('should recognize complex types', () => {
        class Foo {}

        const complexTypes = [
            {}, // Standard
            { a: 1 }, // Standard with value
            new Foo(), // Instance of class
            Object.create({}), // Object without prototype
            Object.create(null), // Empty, raw, virgin object
            [], // Array
            [1, 2], // Array wth values
            // function () {}, // Old-style function is forbidden by CI
            () => {}, // Arrow function
        ]

        // Isn't top level
        component.key = 'foo'

        for (const item of complexTypes) {
            component.value = item
            expect(component.isComplex()).toBeTrue()
            expect(component.isPrimitive()).toBeFalse()
        }
    })

    it('should recognize array types', () => {
        // Completely not recommended
        class Foo extends Array {}

        const arrayTypes = [
            [], // Empty array
            [1, 2, 3], // Array with values
            new Foo(), // Custom subclass
        ]

        // Isn't root level
        component.key = 'foo'

        for (const item of arrayTypes) {
            component.value = item
            expect(component.isArray()).toBeTrue()

            fixture.detectChanges()
            const element = fixture.debugElement.query(By.css('.tree-level__value--array'))
            expect(element).not.toBeNull()
        }
    })

    it('should recognize strings', () => {
        const strings = [
            '', // Empty
            'foobar', // Standard
            ' ', // Whitespace
            '\r', // Caret
        ]

        for (const item of strings) {
            component.value = item
            expect(component.isString()).toBeTrue()

            fixture.detectChanges()
            const element = fixture.debugElement.query(By.css('.tree-level__value--string'))
            expect(element).not.toBeNull()
        }
    })

    it('should recognize numbers', () => {
        const numbers = [
            0, // Zero
            42, // Integer
            -1, // Negative
            3.14, // Float
            10 ** 100, // Big integer
            parseFloat('Infinity'), // Infinity
            parseFloat('NaN'), // Not a number, but still number
        ]

        for (const item of numbers) {
            component.value = item
            expect(component.isNumber()).toBeTrue()
        }
    })

    it('should recognize empty objects', () => {
        class Foo {}

        const empties = [
            [{}, 'object'], // Object
            [[], 'array'], // Array
            [new Foo(), 'object'], // Empty instance
        ]

        for (const [item, kind] of empties) {
            component.value = item
            expect(component.isEmpty()).toBeTrue()

            fixture.detectChanges()
            const root = fixture.debugElement.query(By.css('.tree-level--leaf'))
            expect(root).not.toBeNull()
            const classSelector = kind === 'object' ? '.tree-level__value--object' : '.tree-level__value--array'
            const value = root.query(By.css(classSelector))
            expect(value).not.toBeNull()
        }
    })

    it('should distinguish plain object from array', () => {
        component.value = {}
        expect(component.isObject()).toBeTrue()

        component.value = []
        expect(component.isObject()).toBeFalse()
    })

    it('should recognize null or undefined', () => {
        component.value = null
        expect(component.isNullOrUndefined()).toBeTrue()

        component.value = undefined
        expect(component.isNullOrUndefined()).toBeTrue()
    })

    it('should indicate top level', () => {
        const emptyKeys = ['', null, undefined]

        for (const key of emptyKeys) {
            component.key = key
            expect(component.isRootLevel()).toBeTrue()
        }
    })

    it('should indicate when value is not set', () => {
        // Root level, empty value
        component.key = null
        component.value = null
        expect(component.hasAssignedValue()).toBeFalse()

        component.key = null
        component.value = undefined
        expect(component.hasAssignedValue()).toBeFalse()

        // Root level, set value
        component.key = null
        component.value = 42
        expect(component.hasAssignedValue()).toBeTrue()

        // Nested level, empty value
        component.key = 'foo'
        component.value = null
        expect(component.hasAssignedValue()).toBeTrue()

        // Nested level, set value
        component.key = 'foo'
        component.value = 42
        expect(component.hasAssignedValue()).toBeTrue()
    })

    it('should recognize when object is corrupted', () => {
        const corruptedValues = [
            '[1,2,3', // Skipped square bracket
            '1,2,3}', // Skipped curly bracket
            ['', 1, true], // Mishmash array
        ]

        for (const item of corruptedValues) {
            component.value = item
            expect(component.isCorrupted()).toBeTrue()
        }
    })

    it('should recognize that has only one child', () => {
        component.value = { foo: 'bar' }
        expect(component.hasSingleChild()).toBeTrue()

        component.value = ['baz']
        expect(component.hasSingleChild()).toBeTrue()

        component.value = { foo: 'bar', baz: 42 }
        expect(component.hasSingleChild()).toBeFalse()

        component.value = {}
        expect(component.hasSingleChild()).toBeFalse()

        component.value = []
        expect(component.hasSingleChild()).toBeFalse()

        component.value = 42
        expect(component.hasSingleChild()).toBeFalse()
    })
})
