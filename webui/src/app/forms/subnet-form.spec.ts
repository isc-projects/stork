import { DaemonGroup } from './subnet-form'

describe('SubnetForm', () => {
    it('should construct a daemon group label properly', () => {
        expect(new DaemonGroup(42, []).label).toBe('empty')
        expect(new DaemonGroup(42, [{ id: 1, label: 'name', serverTag: 'tag' }]).label).toBe('name')
        expect(
            new DaemonGroup(42, [{ id: 1, label: 'name', serverTag: 'tag', hooks: ['libdhcp_cb_cmds.so'] }]).label
        ).toBe('tag (1 daemon)')
        expect(
            new DaemonGroup(42, [
                { id: 1, label: 'foo', serverTag: 'tag' },
                { id: 2, label: 'bar', serverTag: 'tag' },
            ]).label
        ).toBe('tag (2 daemons)')
    })
})
