import { LocaltimePipe } from './localtime.pipe'
import { datetimeToLocal } from '../utils'

describe('LocaltimePipe', () => {
    it('create an instance', () => {
        const pipe = new LocaltimePipe()
        expect(pipe).toBeTruthy()
    })

    it('should convert epoch time to local time', () => {
        const pipe = new LocaltimePipe()
        const converted = pipe.transform(1616149050)
        const date = new Date(1616149050000)
        expect(converted).toEqual(datetimeToLocal(date))
    })
})
