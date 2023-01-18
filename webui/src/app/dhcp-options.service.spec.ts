import { TestBed } from '@angular/core/testing'

import { DhcpOptionsService } from './dhcp-options.service'
import { stdDhcpv4OptionDefs } from './std-dhcpv4-option-defs'
import { stdDhcpv6OptionDefs } from './std-dhcpv6-option-defs'

describe('DhcpOptionsService', () => {
    let service: DhcpOptionsService

    beforeEach(() => {
        TestBed.configureTestingModule({})
        service = TestBed.inject(DhcpOptionsService)
    })

    it('should be created', () => {
        expect(service).toBeTruthy()
    })

    it('should return all configurable standard DHCPv4 options', () => {
        let options = service.getStandardDhcpv4Options()
        expect(options.length).toBe(98)

        // Validate one of them to make sure they are DHCPv4 options.
        let selectedOption = options.find((o) => o.value === 5)
        expect(selectedOption).toBeTruthy()
        expect(selectedOption.label).toBe('(5) Name Server')
    })

    it('should return all configurable standard DHCPv6 options', () => {
        let options = service.getStandardDhcpv6Options()
        expect(options.length).toBe(56)

        // Validate one of them to make sure they are DHCPv6 options.
        let selectedOption = options.find((o) => o.value === 23)
        expect(selectedOption).toBeTruthy()
        expect(selectedOption.label).toBe('(23) OPTION_DNS_SERVERS')
    })

    it('should return selected standard DHCPv4 option', () => {
        let option = service.findStandardDhcpv4Option(42)
        expect(option).toBeTruthy()
        expect(option.value).toBe(42)
        expect(option.label).toBe('(42) NTP Servers')

        // Non existing option.
        expect(service.findStandardDhcpv4Option(1024)).toBeFalsy()
    })

    it('should return selected standard DHCPv6 option', () => {
        let option = service.findStandardDhcpv6Option(66)
        expect(option).toBeTruthy()
        expect(option.value).toBe(66)
        expect(option.label).toBe('(66) OPTION_RSOO')

        // Non existing option.
        expect(service.findStandardDhcpv6Option(1024)).toBeFalsy()
    })

    it('should return standard DHCPv4 option definition', () => {
        let defs = stdDhcpv4OptionDefs
        for (let def of defs) {
            let foundDef = service.findStandardDhcpv4OptionDef(def.code, def.space)
            expect(foundDef).toBeTruthy()
            expect(foundDef).toEqual(def)
        }
    })

    it('should return standard DHCPv6 option definition', () => {
        let defs = stdDhcpv6OptionDefs
        for (let def of defs) {
            let foundDef = service.findStandardDhcpv6OptionDef(def.code, def.space)
            expect(foundDef).toBeTruthy()
            expect(foundDef).toEqual(def)
        }
    })

    it('should return standard DHCPv4 option definitions by space', () => {
        let defs = stdDhcpv4OptionDefs
        let foundDefs = service.findStandardDhcpv4OptionDefsBySpace(null)
        expect(foundDefs.length).toBe(defs.filter((def) => def.space === 'dhcp4').length)

        foundDefs = service.findStandardDhcpv4OptionDefsBySpace('dhcp-agent-options-space')
        expect(foundDefs.length).toBe(defs.filter((def) => def.space === 'dhcp-agent-options-space').length)
    })

    it('should return standard DHCPv6 option definitions by space', () => {
        let defs = stdDhcpv6OptionDefs
        let foundDefs = service.findStandardDhcpv6OptionDefsBySpace(null)
        expect(foundDefs.length).toBe(defs.filter((def) => def.space === 'dhcp6').length)

        // Go over other option spaces.
        let spaces = ['s46-cont-mape-options', 's46-cont-mapt-options', 's46-cont-lw-options', 's46-v4v6bind-options']
        for (let space of spaces) {
            foundDefs = service.findStandardDhcpv6OptionDefsBySpace(space)
            expect(foundDefs.length).toBe(defs.filter((def) => def.space === space).length)
        }
    })
})
