import { convertSortingFields, SortDir } from './table'
import { UserSortField } from './backend'
import { TableLazyLoadEvent } from 'primeng/table'

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
})
