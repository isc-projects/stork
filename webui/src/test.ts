// This file is required by karma.conf.js and loads recursively all the .spec and framework files

import 'zone.js/testing'
import { provideZoneChangeDetection } from '@angular/core'
import { getTestBed, TestBed, TestModuleMetadata } from '@angular/core/testing'
import { BrowserDynamicTestingModule, platformBrowserDynamicTesting } from '@angular/platform-browser-dynamic/testing'

// First, initialize the Angular testing environment.
getTestBed().initTestEnvironment(BrowserDynamicTestingModule, platformBrowserDynamicTesting(), {
    errorOnUnknownElements: true,
    errorOnUnknownProperties: true,
})

// Angular 21 defaults tests to zoneless change detection. Zone.js-based apps must opt in explicitly.
const configureTestingModule = TestBed.configureTestingModule.bind(TestBed)
TestBed.configureTestingModule = (moduleDef: TestModuleMetadata) =>
    configureTestingModule({
        ...moduleDef,
        providers: [provideZoneChangeDetection(), ...(moduleDef.providers ?? [])],
    })
