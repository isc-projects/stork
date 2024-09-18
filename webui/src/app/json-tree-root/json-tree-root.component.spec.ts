import { HttpClientTestingModule } from '@angular/common/http/testing'

import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { By } from '@angular/platform-browser'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { MessageService } from 'primeng/api'
import { of } from 'rxjs'
import { AuthService } from '../auth.service'
import { UsersService } from '../backend'
import { JsonTreeComponent } from '../json-tree/json-tree.component'

import { JsonTreeRootComponent } from './json-tree-root.component'
import { RouterModule } from '@angular/router'

describe('JsonTreeRootComponent', () => {
    let component: JsonTreeRootComponent
    let fixture: ComponentFixture<JsonTreeRootComponent>
    let authService: AuthService
    let userService: UsersService

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [HttpClientTestingModule, NoopAnimationsModule, RouterModule],
            declarations: [JsonTreeRootComponent, JsonTreeComponent],
            providers: [MessageService, UsersService],
        }).compileComponents()
        userService = TestBed.inject(UsersService)
        authService = TestBed.inject(AuthService)
    }))

    beforeEach(() => {
        spyOn(userService, 'createSession').and.returnValues(
            of({
                id: 1,
                login: 'foo',
                email: 'foo@bar.baz',
                name: 'foo',
                lastname: 'bar',
                groups: [],
            } as any),
            of({
                id: 1,
                login: 'foo',
                email: 'foo@bar.baz',
                name: 'foo',
                lastname: 'bar',
                groups: [1],
            } as any)
        )

        authService.login('boz', 'foo', 'bar', '/')

        fixture = TestBed.createComponent(JsonTreeRootComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('standard user should not show the secrets', () => {
        expect(component.canShowSecrets).toBeFalse()
    })

    it('admin should not show the secrets', async () => {
        authService.logout()
        authService.login('boz', 'foo', 'bar', 'baz')
        await fixture.whenStable()

        expect(component.canShowSecrets).toBeTrue()
    })

    it('should calculate a valid number of nodes to expand', () => {
        component.autoExpand = 'all'
        expect(component.autoExpandNodeCount).toBe(Number.MAX_SAFE_INTEGER)

        component.autoExpand = 'none'
        expect(component.autoExpandNodeCount).toBe(0)

        component.autoExpand = 42
        expect(component.autoExpandNodeCount).toBe(42)
    })

    it('should pass through all arguments', () => {
        component.autoExpand = 42
        component.value = 'foo'
        component.secretKeys = ['token']

        fixture.detectChanges()

        const jsonElement = fixture.debugElement.query(By.directive(JsonTreeComponent))
        expect(jsonElement).toBeDefined()
        const jsonComponent = jsonElement.componentInstance as JsonTreeComponent

        expect(jsonComponent.value).toBe('foo')
        expect(jsonComponent.autoExpandMaxNodeCount).toBe(42)
        expect(jsonComponent.secretKeys).toEqual(['token'])
    })

    it('should initialize the inner component with correct defaults', async () => {
        authService.logout()
        authService.login('boz', 'foo', 'bar', 'baz')
        await fixture.whenStable()
        fixture.detectChanges()

        const jsonElement = fixture.debugElement.query(By.directive(JsonTreeComponent))
        expect(jsonElement).toBeDefined()
        const jsonComponent = jsonElement.componentInstance as JsonTreeComponent

        expect(jsonComponent.key).toBeNull()
        expect(jsonComponent.recursionLevel).toBe(0)
        expect(jsonComponent.canShowSecrets).toBeTrue()
        expect(jsonComponent.isRootLevel()).toBeTrue()
    })
})
