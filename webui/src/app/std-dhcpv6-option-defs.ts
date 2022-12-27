/**
 * Attention! Generated Code!
 *
 * Run "rake gen:option_defs" to regenerate the option definitions
 * specified in the "codegen/std_dhcpv6_option_def.json" using the
 * template file "std-dhcpv6-option-defs.ts.template" into the
 * "std-dhcpv6-option-defs.ts".
 */

export const stdDhcpv6OptionDefs = [
    {
        code: 94,
        encapsulate: 's46-cont-mape-options',
        name: 's46-cont-mape',
        space: 'dhcp6',
        type: 'empty',
    },
    {
        code: 95,
        encapsulate: 's46-cont-mapt-options',
        name: 's46-cont-mapt',
        space: 'dhcp6',
        type: 'empty',
    },
    {
        code: 96,
        encapsulate: 's46-cont-lw-options',
        name: 's46-cont-lw',
        space: 'dhcp6',
        type: 'empty',
    },
    {
        code: 90,
        encapsulate: '',
        name: 's46-br',
        space: 's46-cont-mape-options',
        type: 'ipv6-address',
    },
    {
        code: 89,
        encapsulate: 's46-rule-options',
        name: 's46-rule',
        recordTypes: ['uint8', 'uint8', 'uint8', 'ipv4-address', 'ipv6-prefix'],
        space: 's46-cont-mape-options',
        type: 'record',
    },
    {
        code: 89,
        encapsulate: 's46-rule-options',
        name: 's46-rule',
        recordTypes: ['uint8', 'uint8', 'uint8', 'ipv4-address', 'ipv6-prefix'],
        space: 's46-cont-mapt-options',
        type: 'record',
    },
    {
        code: 91,
        encapsulate: '',
        name: 's46-dmr',
        space: 's46-cont-mapt-options',
        type: 'ipv6-prefix',
    },
    {
        code: 90,
        encapsulate: '',
        name: 's46-br',
        space: 's46-cont-lw-options',
        type: 'ipv6-address',
    },
    {
        code: 92,
        encapsulate: 's46-v4v6bind-options',
        name: 's46-v4v6bind',
        recordTypes: ['ipv4-address', 'ipv6-prefix'],
        space: 's46-cont-lw-options',
        type: 'record',
    },
    {
        code: 93,
        encapsulate: '',
        name: 's46-portparams',
        recordTypes: ['uint8', 'psid'],
        space: 's46-rule-options',
        type: 'record',
    },
    {
        code: 93,
        encapsulate: '',
        name: 's46-portparams',
        recordTypes: ['uint8', 'psid'],
        space: 's46-v4v6bind-options',
        type: 'record',
    },
]
