import { getSeverity, getTooltip } from './zone-inventory-utils'
import { ZoneInventoryState } from './backend'
import StatusEnum = ZoneInventoryState.StatusEnum

describe('ZoneInventoryUtils', () => {
    it('should get severity', () => {
        // Arrange + Act + Assert
        expect(getSeverity(StatusEnum.Busy)).toEqual('warn')
        expect(getSeverity(StatusEnum.Ok)).toEqual('success')
        expect(getSeverity(StatusEnum.Erred)).toEqual('danger')
        expect(getSeverity(StatusEnum.Uninitialized)).toEqual('secondary')
        expect(getSeverity(<StatusEnum>'foo')).toEqual('info')
    })

    it('should get tooltip', () => {
        // Arrange + Act + Assert
        expect(getTooltip(StatusEnum.Busy)).toContain('Zone inventory on the agent is busy')
        expect(getTooltip(StatusEnum.Ok)).toContain('successfully fetched all zones')
        expect(getTooltip(StatusEnum.Erred)).toContain('Error when communicating with a zone inventory')
        expect(getTooltip(StatusEnum.Uninitialized)).toContain('Zone inventory on the agent was not initialized')
        expect(getTooltip(<StatusEnum>'foo')).toBeNull()
    })
})
