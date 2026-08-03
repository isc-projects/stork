import { convertSortingFields, hasFilter } from './table'
import { SortDir, UserSortField } from './backend'
import { TableLazyLoadEvent } from 'primeng/table'
import { FilterMetadata } from 'primeng/api'

describe('Table', () => {
    it('should convert sorting fields', () => {
        let testEvent: TableLazyLoadEvent = {
            sortOrder: undefined,
        }
        let answer = convertSortingFields<UserSortField>(undefined)
        expect(answer).toEqual([null, null])

        answer = convertSortingFields<UserSortField>(testEvent)
        expect(answer).toEqual([null, null])

        testEvent = {
            sortOrder: undefined,
            sortField: undefined,
        }
        answer = convertSortingFields<UserSortField>(testEvent)
        expect(answer).toEqual([null, null])

        testEvent = {
            sortOrder: undefined,
            sortField: UserSortField.Name,
        }
        answer = convertSortingFields<UserSortField>(testEvent)
        expect(answer).toEqual([UserSortField.Name, null])

        testEvent = {
            sortOrder: -1,
            sortField: UserSortField.Name,
        }
        answer = convertSortingFields<UserSortField>(testEvent)
        expect(answer).toEqual([UserSortField.Name, SortDir.Desc])

        testEvent = {
            sortOrder: 0,
            sortField: UserSortField.Name,
        }
        answer = convertSortingFields<UserSortField>(testEvent)
        expect(answer).toEqual([UserSortField.Name, SortDir.Asc])

        testEvent = {
            sortOrder: 1,
            sortField: UserSortField.Name,
        }
        answer = convertSortingFields<UserSortField>(testEvent)
        expect(answer).toEqual([UserSortField.Name, SortDir.Asc])
    })

    it('should check blank or non-blank filters', () => {
        const f1: { [p: string]: FilterMetadata } = {
            boolean: {
                value: false,
                matchMode: 'contains', // "contains" matchMode makes bool "false" values considered blank filter
            },
            'tri-state': {
                value: null,
                matchMode: 'equals', // "equals" matchMode filter is considered blank when its value is strictly equal to null
            },
            text: {
                value: '',
                matchMode: 'contains', // "contains" matchMode makes blank string considered blank filter
            },
        }
        expect(hasFilter(f1)).toBeFalse()

        const f2: { [p: string]: FilterMetadata } = {
            boolean: {
                value: false,
                matchMode: 'contains',
            },
            'tri-state': {
                value: false,
                matchMode: 'equals', // "equals" matchMode makes bool "false" values considered non-blank filters
            },
            text: {
                value: '',
                matchMode: 'contains',
            },
        }
        expect(hasFilter(f2)).toBeTrue()

        const f3: { [p: string]: FilterMetadata } = {
            boolean: {
                value: true,
                matchMode: 'contains',
            },
            'tri-state': {
                value: null,
                matchMode: 'equals',
            },
            text: {
                value: '',
                matchMode: 'contains',
            },
        }
        expect(hasFilter(f3)).toBeTrue()

        const f4: { [p: string]: FilterMetadata } = {
            boolean: {
                value: false,
                matchMode: 'contains',
            },
            'tri-state': {
                value: null,
                matchMode: 'equals',
            },
            text: {
                value: ' ',
                matchMode: 'contains',
            },
        }
        expect(hasFilter(f4)).toBeTrue()
    })
})
