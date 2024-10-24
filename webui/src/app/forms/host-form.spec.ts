import { UntypedFormBuilder } from '@angular/forms'
import { HostForm } from './host-form'

describe('HostForm', () => {
    let fb: UntypedFormBuilder
    let form: HostForm

    beforeEach(() => {
        fb = new UntypedFormBuilder()
        form = new HostForm()
        form.group = fb.group({
            selectedSubnet: [''],
        })
    })

    it('Returns daemon by ID', () => {
        form.allDaemons = [
            {
                id: 1,
                appId: 1,
                appType: 'kea',
                name: 'dhcp4',
                version: '3.0.0',
                label: 'server1',
            },
            {
                id: 2,
                appId: 3,
                appType: 'bind9',
                name: 'named',
                version: '3.0.0',
                label: 'server2',
            },
        ]

        let daemon = form.getDaemonById(1)
        expect(daemon).toBeTruthy()
        expect(daemon.id).toBe(1)
        expect(daemon.appId).toBe(1)
        expect(daemon.name).toBe('dhcp4')
        expect(daemon.label).toBe('server1')

        daemon = form.getDaemonById(2)
        expect(daemon).toBeTruthy()
        expect(daemon.id).toBe(2)
        expect(daemon.appId).toBe(3)
        expect(daemon.name).toBe('named')
        expect(daemon.label).toBe('server2')

        expect(form.getDaemonById(3)).toBeFalsy()
    })

    it('Correctly updates form for selected daemons', () => {
        form.allDaemons = [
            {
                id: 1,
                appId: 1,
                appType: 'kea',
                name: 'dhcp4',
                version: '3.0.0',
                label: 'server1',
            },
            {
                id: 2,
                appId: 1,
                appType: 'kea',
                name: 'dhcp6',
                version: '3.0.0',
                label: 'server2',
            },
            {
                id: 3,
                appId: 2,
                appType: 'kea',
                name: 'dhcp4',
                version: '3.0.0',
                label: 'server3',
            },
            {
                id: 4,
                appId: 2,
                appType: 'kea',
                name: 'dhcp6',
                version: '3.0.0',
                label: 'server4',
            },
        ]
        // Select a DHCPv4 daemon. It is not a breaking change because
        // DHCPv4 options are displayed by default.
        let breakingChange = form.updateFormForSelectedDaemons([1])
        expect(breakingChange).toBeFalse()
        expect(form.filteredDaemons.length).toBe(2)
        expect(form.filteredDaemons[0].id).toBe(1)
        expect(form.filteredDaemons[1].id).toBe(3)

        // Add another DHCPv4 daemon to our selection. It is not a breaking
        // change because we were already in the DHCPv4 mode.
        breakingChange = form.updateFormForSelectedDaemons([1, 3])
        expect(breakingChange).toBeFalse()
        expect(form.filteredDaemons.length).toBe(2)
        expect(form.filteredDaemons[0].id).toBe(1)
        expect(form.filteredDaemons[1].id).toBe(3)

        // Reduce selected DHCPv4 daemons. It is not a breaking change.
        breakingChange = form.updateFormForSelectedDaemons([3])
        expect(breakingChange).toBeFalse()
        expect(form.filteredDaemons.length).toBe(2)
        expect(form.filteredDaemons[0].id).toBe(1)
        expect(form.filteredDaemons[1].id).toBe(3)

        // Unselect all daemons. It is a breaking change.
        breakingChange = form.updateFormForSelectedDaemons([])
        expect(breakingChange).toBeTrue()
        expect(form.filteredDaemons.length).toBe(4)
        expect(form.filteredDaemons[0].id).toBe(1)
        expect(form.filteredDaemons[1].id).toBe(2)
        expect(form.filteredDaemons[2].id).toBe(3)
        expect(form.filteredDaemons[3].id).toBe(4)

        // Select DHCPv6 daemon. It is a breaking change because by default
        // we display DHCPv4 options.
        breakingChange = form.updateFormForSelectedDaemons([2])
        expect(breakingChange).toBeTrue()
        expect(form.filteredDaemons.length).toBe(2)
        expect(form.filteredDaemons[0].id).toBe(2)
        expect(form.filteredDaemons[1].id).toBe(4)

        // Select another DHCPv6 daemon. It is not a breaking change.
        breakingChange = form.updateFormForSelectedDaemons([2])
        expect(breakingChange).toBeFalse()
        expect(form.filteredDaemons.length).toBe(2)
        expect(form.filteredDaemons[0].id).toBe(2)
        expect(form.filteredDaemons[1].id).toBe(4)

        // Unselect DHCPv6 daemons.
        breakingChange = form.updateFormForSelectedDaemons([])
        expect(breakingChange).toBeTrue()
        expect(form.filteredDaemons.length).toBe(4)
        expect(form.filteredDaemons[0].id).toBe(1)
        expect(form.filteredDaemons[1].id).toBe(2)
        expect(form.filteredDaemons[2].id).toBe(3)
        expect(form.filteredDaemons[3].id).toBe(4)
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
