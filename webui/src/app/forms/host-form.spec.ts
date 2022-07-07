import { FormBuilder } from '@angular/forms'
import { HostForm } from './host-form'

describe('HostForm', () => {
    let fb: FormBuilder
    let form: HostForm

    beforeEach(() => {
        fb = new FormBuilder()
        form = new HostForm()
        form.group = fb.group({
            selectedSubnet: [''],
        })
    })

    it('Returns correct ipv4 selected subnet range', () => {
        form.filteredSubnets = [
            {
                id: 1,
                subnet: '192.0.2.0/24',
            },
            {
                id: 2,
                subnet: '10.0.0.0/8',
            },
        ]
        form.group.get('selectedSubnet').setValue(1)
        let range = form.getSelectedSubnetRange()
        expect(range).toBeTruthy()
        expect(range.length).toBe(2)
        expect(range[0]).toBe('192.0.2.0/24')
        expect(range[1]).toBeTruthy()
        expect(range[1].getFirst().toString()).toBe('192.0.2.0')
        expect(range[1].getLast().toString()).toBe('192.0.2.255')

        form.group.get('selectedSubnet').setValue(2)
        range = form.getSelectedSubnetRange()
        expect(range).toBeTruthy()
        expect(range.length).toBe(2)
        expect(range[0]).toBe('10.0.0.0/8')
        expect(range[1]).toBeTruthy()
        expect(range[1].getFirst().toString()).toBe('10.0.0.0')
        expect(range[1].getLast().toString()).toBe('10.255.255.255')

        // Non existing subnet should cause the function to return null.
        form.group.get('selectedSubnet').setValue(3)
        range = form.getSelectedSubnetRange()
        expect(range).toBeFalsy()
    })

    it('Returns correct ipv6 selected subnet range', () => {
        form.filteredSubnets = [
            {
                id: 1,
                subnet: '2001:db8:1::/64',
            },
            {
                id: 2,
                subnet: '3000:1::/32',
            },
        ]
        form.group.get('selectedSubnet').setValue(1)
        let range = form.getSelectedSubnetRange()
        expect(range).toBeTruthy()
        expect(range.length).toBe(2)
        expect(range[0]).toBe('2001:db8:1::/64')
        expect(range[1]).toBeTruthy()
        expect(range[1].getFirst().toString()).toBe('2001:db8:1:0:0:0:0:0')
        expect(range[1].getLast().toString()).toBe('2001:db8:1:0:ffff:ffff:ffff:ffff')

        form.group.get('selectedSubnet').setValue(2)
        range = form.getSelectedSubnetRange()
        expect(range).toBeTruthy()
        expect(range.length).toBe(2)
        expect(range[0]).toBe('3000:1::/32')
        expect(range[1]).toBeTruthy()
        expect(range[1].getFirst().toString()).toBe('3000:1:0:0:0:0:0:0')
        expect(range[1].getLast().toString()).toBe('3000:1:ffff:ffff:ffff:ffff:ffff:ffff')

        // Non existing subnet should cause the function to return null.
        form.group.get('selectedSubnet').setValue(3)
        range = form.getSelectedSubnetRange()
        expect(range).toBeFalsy()
    })
})
