import { Injectable } from '@angular/core'
import { DhcpOptionDef } from './dhcp-option-def'
import { stdDhcpv4OptionDefs } from './std-dhcpv4-option-defs'
import { stdDhcpv6OptionDefs } from './std-dhcpv6-option-defs'

/**
 * An interface to a DHCP option description.
 *
 * It is used to define a list of standard DHCP options.
 */
export interface DhcpOptionListItem {
    label: string
    value: number
}

/**
 * A service exposing a list of DHCP options with their mapping between
 * option codes and friendly names.
 *
 * A full list of options returned by this service can be used in the
 * forms containing a list of available options. In the componwnents
 * that display configured options it is useful to find option name by
 * the option code value. This service provides such a capability.
 */
@Injectable({
    providedIn: 'root',
})
export class DhcpOptionsService {
    /**
     * Defines a list of the configurable standard DHCPv4 options.
     *
     * Commented out options are not configurable by a user.
     */
    private _dhcpv4Options: DhcpOptionListItem[] = [
        /* {
            label: '(1) Subnet Mask',
            value: 1,
        }, */
        {
            label: '(2) Time Offset',
            value: 2,
        },
        {
            label: '(3) Router',
            value: 3,
        },
        {
            label: '(4) Time Server',
            value: 4,
        },
        {
            label: '(5) Name Server',
            value: 5,
        },
        {
            label: '(6) Domain Server',
            value: 6,
        },
        {
            label: '(7) Log Server',
            value: 7,
        },
        {
            label: '(8) Quotes Server',
            value: 8,
        },
        {
            label: '(9) LPR Server',
            value: 9,
        },
        {
            label: '(10) Impress Server',
            value: 10,
        },
        {
            label: '(11) RLP Server',
            value: 11,
        },
        /* {
            label: '(12) Hostname',
            value: 12,
        }, */
        {
            label: '(13) Boot File Size',
            value: 13,
        },
        {
            label: '(14) Merit Dump File',
            value: 14,
        },
        {
            label: '(15) Domain Name',
            value: 15,
        },
        {
            label: '(16) Swap Server',
            value: 16,
        },
        {
            label: '(17) Root Path',
            value: 17,
        },
        {
            label: '(18) Extension File',
            value: 18,
        },
        {
            label: '(19) Forward On/Off',
            value: 19,
        },
        {
            label: '(20) SrcRte On/Off',
            value: 20,
        },
        {
            label: '(21) Policy Filter',
            value: 21,
        },
        {
            label: '(22) Max DG Assembly',
            value: 22,
        },
        {
            label: '(23) Default IP TTL',
            value: 23,
        },
        {
            label: '(24) MTU Timeout',
            value: 24,
        },
        {
            label: '(25) MTU Plateau',
            value: 25,
        },
        {
            label: '(26) MTU Interface',
            value: 26,
        },
        {
            label: '(27) MTU Subnet',
            value: 27,
        },
        {
            label: '(28) Broadcast Address',
            value: 28,
        },
        {
            label: '(29) Mask Discovery',
            value: 29,
        },
        {
            label: '(30) Mask Supplier',
            value: 30,
        },
        {
            label: '(31) Router Discovery',
            value: 31,
        },
        {
            label: '(32) Router Request',
            value: 32,
        },
        {
            label: '(33) Static Route',
            value: 33,
        },
        {
            label: '(34) Trailers',
            value: 34,
        },
        {
            label: '(35) ARP Timeout',
            value: 35,
        },
        {
            label: '(36) Ethernet',
            value: 36,
        },
        {
            label: '(37) Default TCP TTL',
            value: 37,
        },
        {
            label: '(38) Keepalive Time',
            value: 38,
        },
        {
            label: '(39) Keepalive Data',
            value: 39,
        },
        {
            label: '(40) NIS Domain',
            value: 40,
        },
        {
            label: '(41) NIS Servers',
            value: 41,
        },
        {
            label: '(42) NTP Servers',
            value: 42,
        },
        {
            label: '(43) Vendor Specific',
            value: 43,
        },
        {
            label: '(44) NETBIOS Name Srv',
            value: 44,
        },
        {
            label: '(45) NETBIOS Dist Srv',
            value: 45,
        },
        {
            label: '(46) NETBIOS Node Type',
            value: 46,
        },
        {
            label: '(47) NETBIOS Scope',
            value: 47,
        },
        {
            label: '(48) X Window Font',
            value: 48,
        },
        {
            label: '(49) X Window Manager',
            value: 49,
        },
        /*{
            label: '(50) Address Request',
            value: 50,
        },
        {
            label: '(51) Address Time',
            value: 51,
        },*/
        {
            label: '(52) Overload',
            value: 52,
        },
        /*{
            label: '(53) DHCP Msg Type',
            value: 53,
        },*/
        {
            label: '(54) DHCP Server Id',
            value: 54,
        },
        /*{
            label: '(55) Parameter List',
            value: 55,
        },*/
        {
            label: '(56) DHCP Message',
            value: 56,
        },
        {
            label: '(57) DHCP Max Msg Size',
            value: 57,
        },
        /*{
            label: '(58) Renewal Time',
            value: 58,
        },
        {
            label: '(59) Rebinding Time',
            value: 59,
        },*/
        {
            label: '(60) Class Id',
            value: 60,
        },
        /*{
            label: '(61) Client Id',
            value: 61,
        },*/
        {
            label: '(62) NetWare/IP Domain',
            value: 62,
        },
        {
            label: '(63) NetWare/IP Option',
            value: 63,
        },
        {
            label: '(64) NIS-Domain-Name',
            value: 64,
        },
        {
            label: '(65) NIS-Server-Addr',
            value: 65,
        },
        {
            label: '(66) Server-Name',
            value: 66,
        },
        {
            label: '(67) Bootfile-Name',
            value: 67,
        },
        {
            label: '(68) Home-Agent-Addrs',
            value: 68,
        },
        {
            label: '(69) SMTP-Server',
            value: 69,
        },
        {
            label: '(70) POP3-Server',
            value: 70,
        },
        {
            label: '(71) NNTP-Server',
            value: 71,
        },
        {
            label: '(72) WWW-Server',
            value: 72,
        },
        {
            label: '(73) Finger-Server',
            value: 73,
        },
        {
            label: '(74) IRC-Server',
            value: 74,
        },
        {
            label: '(75) StreetTalk-Server',
            value: 75,
        },
        {
            label: '(76) STDA-Server',
            value: 76,
        },
        {
            label: '(77) User-Class',
            value: 77,
        },
        {
            label: '(78) Directory Agent',
            value: 78,
        },
        {
            label: '(79) Service Scope',
            value: 79,
        },
        /*{
            label: '(80) Rapid Commit',
            value: 80,
        },
        {
            label: '(81) Client FQDN',
            value: 81,
        },
        {
            label: '(82) Relay Agent Information',
            value: 82,
        },
        {
            label: '(83) iSNS',
            value: 83,
        },
        {
            label: '(84) REMOVED/Unassigned',
            value: 84,
        },*/
        {
            label: '(85) NDS Servers',
            value: 85,
        },
        {
            label: '(86) NDS Tree Name',
            value: 86,
        },
        {
            label: '(87) NDS Context',
            value: 87,
        },
        {
            label: '(88) BCMCS Controller Domain Name list',
            value: 88,
        },
        {
            label: '(89) BCMCS Controller IPv(4) address option',
            value: 89,
        },
        /*{
            label: '(90) Authentication',
            value: 90,
        },
        {
            label: '(91) client-last-transaction-time option',
            value: 91,
        },
        {
            label: '(92) associated-ip option',
            value: 92,
        },*/
        {
            label: '(93) Client System',
            value: 93,
        },
        {
            label: '(94) Client NDI',
            value: 94,
        },
        /*{
            label: '(95) LDAP',
            value: 95,
        },
        {
            label: '(96) REMOVED/Unassigned',
            value: 96,
        },*/
        {
            label: '(97) UUID/GUID',
            value: 97,
        },
        {
            label: '(98) User-Auth',
            value: 98,
        },
        {
            label: '(99) GEOCONF_CIVIC',
            value: 99,
        },
        {
            label: '(100) PCode',
            value: 100,
        },
        {
            label: '(101) TCode',
            value: 101,
        },
        /*{
            label: '(102-107) REMOVED/Unassigned',
            value: 102 - 107,
        },*/
        {
            label: '(108) IPv6-Only Preferred',
            value: 108,
        },
        /*{
            label: '(109) OPTION_DHCP4O6_S46_SADDR',
            value: 109,
        },
        {
            label: '(110) REMOVED/Unassigned',
            value: 110,
        },
        {
            label: '(111) Unassigned',
            value: 111,
        },*/
        {
            label: '(112) Netinfo Address',
            value: 112,
        },
        {
            label: '(113) Netinfo Tag',
            value: 113,
        },
        {
            label: '(114) DHCP Captive-Portal',
            value: 114,
        },
        /*{
            label: '(115) REMOVED/Unassigned',
            value: 115,
        },*/
        {
            label: '(116) Auto-Config',
            value: 116,
        },
        {
            label: '(117) Name Service Search',
            value: 117,
        },
        /*{
            label: '(118) Subnet Selection Option',
            value: 118,
        },*/
        {
            label: '(119) Domain Search',
            value: 119,
        },
        /*{
            label: '(120) SIP Servers DHCP Option',
            value: 120,
        },
        {
            label: '(121) Classless Static Route Option',
            value: 121,
        },
        {
            label: '(122) CCC',
            value: 122,
        },
        {
            label: '(123) GeoConf Option',
            value: 123,
        },*/
        {
            label: '(124) V-I Vendor Class',
            value: 124,
        },
        {
            label: '(125) V-I Vendor-Specific Information',
            value: 125,
        },
        /*{
            label: '(126) Removed/Unassigned',
            value: 126,
        },
        {
            label: '(127) Removed/Unassigned',
            value: 127,
        },
        {
            label: '(128) PXE - undefined (vendor specific)',
            value: 128,
        },
        {
            label: '(128) Etherboot signature. 6 bytes: E4:45:74:68:00:00',
            value: 128,
        },
        {
            label: '(128) DOCSIS full security server IP address',
            value: 128,
        },
        {
            label: '(128) TFTP Server IP address (for IP Phone software load)',
            value: 128,
        },
        {
            label: '(129) PXE - undefined (vendor specific)',
            value: 129,
        },
        {
            label: '(129) Kernel options. Variable length string',
            value: 129,
        },
        {
            label: '(129) Call Server IP address',
            value: 129,
        },
        {
            label: '(130) PXE - undefined (vendor specific)',
            value: 130,
        },
        {
            label: '(130) Ethernet interface. Variable length string.',
            value: 130,
        },
        {
            label: '(130) Discrimination string (to identify vendor)',
            value: 130,
        },
        {
            label: '(131) PXE - undefined (vendor specific)',
            value: 131,
        },
        {
            label: '(131) Remote statistics server IP address',
            value: 131,
        },
        {
            label: '(132) PXE - undefined (vendor specific)',
            value: 132,
        },
        {
            label: '(132) IEEE 802.1Q VLAN ID',
            value: 132,
        },
        {
            label: '(133) PXE - undefined (vendor specific)',
            value: 133,
        },
        {
            label: '(133) IEEE 802.1D/p Layer 2 Priority',
            value: 133,
        },
        {
            label: '(134) PXE - undefined (vendor specific)',
            value: 134,
        },
        {
            label: '(134) Diffserv Code Point (DSCP) for VoIP signalling and media streams',
            value: 134,
        },
        {
            label: '(135) PXE - undefined (vendor specific)',
            value: 135,
        },
        {
            label: '(135) HTTP Proxy for phone-specific applications',
            value: 135,
        },*/
        {
            label: '(136) OPTION_PANA_AGENT',
            value: 136,
        },
        {
            label: '(137) OPTION_V4_LOST',
            value: 137,
        },
        {
            label: '(138) OPTION_CAPWAP_AC_V4',
            value: 138,
        },
        /*{
            label: '(139) OPTION-IPv4_Address-MoS',
            value: 139,
        },
        {
            label: '(140) OPTION-IPv4_FQDN-MoS',
            value: 140,
        },*/
        {
            label: '(141) SIP UA Configuration Service Domains',
            value: 141,
        },
        /*{
            label: '(142) OPTION-IPv4_Address-ANDSF',
            value: 142,
        },
        {
            label: '(143) OPTION_V4_SZTP_REDIRECT',
            value: 143,
        },
        {
            label: '(144) GeoLoc',
            value: 144,
        },
        {
            label: '(145) FORCERENEW_NONCE_CAPABLE',
            value: 145,
        },*/
        {
            label: '(146) RDNSS Selection',
            value: 146,
        },
        /*{
            label: '(147) OPTION_V4_DOTS_RI',
            value: 147,
        },
        {
            label: '(148) OPTION_V4_DOTS_ADDRESS',
            value: 148,
        },
        {
            label: '(149) Unassigned',
            value: 149,
        },
        {
            label: '(150) TFTP server address',
            value: 150,
        },
        {
            label: '(150) Etherboot',
            value: 150,
        },
        {
            label: '(150) GRUB configuration path name',
            value: 150,
        },
        {
            label: '(151) status-code',
            value: 151,
        },
        {
            label: '(152) base-time',
            value: 152,
        },
        {
            label: '(153) start-time-of-state',
            value: 153,
        },
        {
            label: '(154) query-start-time',
            value: 154,
        },
        {
            label: '(155) query-end-time',
            value: 155,
        },
        {
            label: '(156) dhcp-state',
            value: 156,
        },
        {
            label: '(157) data-source',
            value: 157,
        },
        {
            label: '(158) OPTION_V(4)_PCP_SERVER',
            value: 158,
        },
        {
            label: '(159) OPTION_V(4)_PORTPARAMS',
            value: 159,
        },
        {
            label: '(160) Unassigned',
            value: 160,
        },
        {
            label: '(161) OPTION_MUD_URL_V4',
            value: 161,
        },
        {
            label: '(162-174) Unassigned',
            value: 162 - 174,
        },
        {
            label: '(175) Etherboot (Tentatively Assigned - 2005-06-23)',
            value: 175,
        },
        {
            label: '(176) IP Telephone (Tentatively Assigned - 2005-06-23)',
            value: 176,
        },
        {
            label: '(177) Etherboot (Tentatively Assigned - 2005-06-23)',
            value: 177,
        },
        {
            label: '(177) PacketCable and CableHome (replaced by 122)',
            value: 177,
        },
        {
            label: '(178-207) Unassigned',
            value: 178 - 207,
        },
        {
            label: '(208) PXELINUX Magic',
            value: 208,
        },
        {
            label: '(209) Configuration File',
            value: 209,
        },
        {
            label: '(210) Path Prefix',
            value: 210,
        },
        {
            label: '(211) Reboot Time',
            value: 211,
        },*/
        {
            label: '(212) OPTION_6RD',
            value: 212,
        },
        {
            label: '(213) OPTION_V4_ACCESS_DOMAIN',
            value: 213,
        },
        /*{
            label: '(214-219) Unassigned',
            value: 214 - 219,
        },
        {
            label: '(220) Subnet Allocation Option',
            value: 220,
        },
        {
            label: '(221) Virtual Subnet Selection (VSS) Option',
            value: 221,
        },
        {
            label: '(222-223) Unassigned',
            value: 222 - 223,
        },
        {
            label: '(224-254) Reserved (Private Use)',
            value: 224 - 254,
        },*/
    ]

    /**
     * Indexes the standard DHCPv4 options by option code for faster lookup.
     */
    private _dhcpv4OptionsByCode: Map<number, DhcpOptionListItem>

    /**
     * Defines a list of configurable standard DHCPv6 options.
     *
     * Commented out options are not configurable by a user.
     */
    private _dhcpv6Options: DhcpOptionListItem[] = [
        /*{
            label: '(0) Reserved',
            value: 0,
        },
        {
            label: '(1) OPTION_CLIENTID',
            value: 1,
        },
        {
            label: '(2) OPTION_SERVERID',
            value: 2,
        },
        {
            label: '(3) OPTION_IA_NA',
            value: 3,
        },
        {
            label: '(4) OPTION_IA_TA',
            value: 4,
        },
        {
            label: '(5) OPTION_IAADDR',
            value: 5,
        },
        {
            label: '(6) OPTION_ORO',
            value: 6,
        },*/
        {
            label: '(7) OPTION_PREFERENCE',
            value: 7,
        },
        /*{
            label: '(8) OPTION_ELAPSED_TIME',
            value: 8,
        },
        {
            label: '(9) OPTION_RELAY_MSG',
            value: 9,
        },
        {
            label: '(10) Unassigned',
            value: 10,
        },
        {
            label: '(11) OPTION_AUTH',
            value: 11,
        },*/
        {
            label: '(12) OPTION_UNICAST',
            value: 12,
        },
        /*{
            label: '(13) OPTION_STATUS_CODE',
            value: 13,
        },
        {
            label: '(14) OPTION_RAPID_COMMIT',
            value: 14,
        },
        {
            label: '(15) OPTION_USER_CLASS',
            value: 15,
        },
        {
            label: '(16) OPTION_VENDOR_CLASS',
            value: 16,
        },
        {
            label: '(17) OPTION_VENDOR_OPTS',
            value: 17,
        },
        {
            label: '(18) OPTION_INTERFACE_ID',
            value: 18,
        },
        {
            label: '(19) OPTION_RECONF_MSG',
            value: 19,
        },
        {
            label: '(20) OPTION_RECONF_ACCEPT',
            value: 20,
        },*/
        {
            label: '(21) OPTION_SIP_SERVER_D',
            value: 21,
        },
        {
            label: '(22) OPTION_SIP_SERVER_A',
            value: 22,
        },
        {
            label: '(23) OPTION_DNS_SERVERS',
            value: 23,
        },
        {
            label: '(24) OPTION_DOMAIN_LIST',
            value: 24,
        },
        /*{
            label: '(25) OPTION_IA_PD',
            value: 25,
        },
        {
            label: '(26) OPTION_IAPREFIX',
            value: 26,
        },*/
        {
            label: '(27) OPTION_NIS_SERVERS',
            value: 27,
        },
        {
            label: '(28) OPTION_NISP_SERVERS',
            value: 28,
        },
        {
            label: '(29) OPTION_NIS_DOMAIN_NAME',
            value: 29,
        },
        {
            label: '(30) OPTION_NISP_DOMAIN_NAME',
            value: 30,
        },
        {
            label: '(31) OPTION_SNTP_SERVERS',
            value: 31,
        },
        {
            label: '(32) OPTION_INFORMATION_REFRESH_TIME',
            value: 32,
        },
        {
            label: '(33) OPTION_BCMCS_SERVER_D',
            value: 33,
        },
        {
            label: '(34) OPTION_BCMCS_SERVER_A',
            value: 34,
        },
        /*{
            label: '(35) Unassigned',
            value: 35,
        },*/
        {
            label: '(36) OPTION_GEOCONF_CIVIC',
            value: 36,
        },
        {
            label: '(37) OPTION_REMOTE_ID',
            value: 37,
        },
        {
            label: '(38) OPTION_SUBSCRIBER_ID',
            value: 38,
        },
        {
            label: '(39) OPTION_CLIENT_FQDN',
            value: 39,
        },
        {
            label: '(40) OPTION_PANA_AGENT',
            value: 40,
        },
        {
            label: '(41) OPTION_NEW_POSIX_TIMEZONE',
            value: 41,
        },
        {
            label: '(42) OPTION_NEW_TZDB_TIMEZONE',
            value: 42,
        },
        {
            label: '(43) OPTION_ERO',
            value: 43,
        },
        {
            label: '(44) OPTION_LQ_QUERY',
            value: 44,
        },
        {
            label: '(45) OPTION_CLIENT_DATA',
            value: 45,
        },
        {
            label: '(46) OPTION_CLT_TIME',
            value: 46,
        },
        {
            label: '(47) OPTION_LQ_RELAY_DATA',
            value: 47,
        },
        {
            label: '(48) OPTION_LQ_CLIENT_LINK',
            value: 48,
        },
        /*{
            label: '(49) OPTION_MIP6_HNIDF',
            value: 49,
        },
        {
            label: '(50) OPTION_MIP6_VDINF',
            value: 50,
        },*/
        {
            label: '(51) OPTION_V6_LOST',
            value: 51,
        },
        {
            label: '(52) OPTION_CAPWAP_AC_V6',
            value: 52,
        },
        {
            label: '(53) OPTION_RELAY_ID',
            value: 53,
        },
        /*{
            label: '(54) OPTION-IPv6_Address-MoS',
            value: 54,
        },
        {
            label: '(55) OPTION-IPv6_FQDN-MoS',
            value: 55,
        },
        {
            label: '(56) OPTION_NTP_SERVER',
            value: 56,
        },*/
        {
            label: '(57) OPTION_V6_ACCESS_DOMAIN',
            value: 57,
        },
        {
            label: '(58) OPTION_SIP_UA_CS_LIST',
            value: 58,
        },
        {
            label: '(59) OPT_BOOTFILE_URL',
            value: 59,
        },
        {
            label: '(60) OPT_BOOTFILE_PARAM',
            value: 60,
        },
        {
            label: '(61) OPTION_CLIENT_ARCH_TYPE',
            value: 61,
        },
        {
            label: '(62) OPTION_NII',
            value: 62,
        },
        /*{
            label: '(63) OPTION_GEOLOCATION',
            value: 63,
        },*/
        {
            label: '(64) OPTION_AFTR_NAME',
            value: 64,
        },
        {
            label: '(65) OPTION_ERP_LOCAL_DOMAIN_NAME',
            value: 65,
        },
        {
            label: '(66) OPTION_RSOO',
            value: 66,
        },
        {
            label: '(67) OPTION_PD_EXCLUDE',
            value: 67,
        },
        /*{
            label: '(68) OPTION_VSS',
            value: 68,
        },
        {
            label: '(69) OPTION_MIP6_IDINF',
            value: 69,
        },
        {
            label: '(70) OPTION_MIP6_UDINF',
            value: 70,
        },
        {
            label: '(71) OPTION_MIP6_HNP',
            value: 71,
        },
        {
            label: '(72) OPTION_MIP6_HAA',
            value: 72,
        },
        {
            label: '(73) OPTION_MIP6_HAF',
            value: 73,
        },*/
        {
            label: '(74) OPTION_RDNSS_SELECTION',
            value: 74,
        },
        /*{
            label: '(75) OPTION_KRB_PRINCIPAL_NAME',
            value: 75,
        },
        {
            label: '(76) OPTION_KRB_REALM_NAME',
            value: 76,
        },
        {
            label: '(77) OPTION_KRB_DEFAULT_REALM_NAME',
            value: 77,
        },
        {
            label: '(78) OPTION_KRB_KDC',
            value: 78,
        },*/
        {
            label: '(79) OPTION_CLIENT_LINKLAYER_ADDR',
            value: 79,
        },
        {
            label: '(80) OPTION_LINK_ADDRESS',
            value: 80,
        },
        /*{
            label: '(81) OPTION_RADIUS',
            value: 81,
        },*/
        {
            label: '(82) OPTION_SOL_MAX_RT',
            value: 82,
        },
        {
            label: '(83) OPTION_INF_MAX_RT',
            value: 83,
        },
        /*{
            label: '(84) OPTION_ADDRSEL',
            value: 84,
        },
        {
            label: '(85) OPTION_ADDRSEL_TABLE',
            value: 85,
        },
        {
            label: '(86) OPTION_V6_PCP_SERVER',
            value: 86,
        },
        {
            label: '(87) OPTION_DHCPV4_MSG',
            value: 87,
        },*/
        {
            label: '(88) OPTION_DHCP4_O_DHCP6_SERVER',
            value: 88,
        },
        {
            label: '(89) OPTION_S46_RULE',
            value: 89,
        },
        {
            label: '(90) OPTION_S46_BR',
            value: 90,
        },
        {
            label: '(91) OPTION_S46_DMR',
            value: 91,
        },
        {
            label: '(92) OPTION_S46_V4V6BIND',
            value: 92,
        },
        {
            label: '(93) OPTION_S46_PORTPARAMS',
            value: 93,
        },
        {
            label: '(94) OPTION_S46_CONT_MAPE',
            value: 94,
        },
        {
            label: '(95) OPTION_S46_CONT_MAPT',
            value: 95,
        },
        {
            label: '(96) OPTION_S46_CONT_LW',
            value: 96,
        },
        /*{
            label: '(97) OPTION_4RD',
            value: 97,
        },
        {
            label: '(98) OPTION_4RD_MAP_RULE',
            value: 98,
        },
        {
            label: '(99) OPTION_4RD_NON_MAP_RULE',
            value: 99,
        },
        {
            label: '(100) OPTION_LQ_BASE_TIME',
            value: 100,
        },
        {
            label: '(101) OPTION_LQ_START_TIME',
            value: 101,
        },
        {
            label: '(102) OPTION_LQ_END_TIME',
            value: 102,
        },*/
        {
            label: '(103) DHCP Captive-Portal',
            value: 103,
        },
        /*{
            label: '(104) OPTION_MPL_PARAMETERS',
            value: 104,
        },
        {
            label: '(105) OPTION_ANI_ATT',
            value: 105,
        },
        {
            label: '(106) OPTION_ANI_NETWORK_NAME',
            value: 106,
        },
        {
            label: '(107) OPTION_ANI_AP_NAME',
            value: 107,
        },
        {
            label: '(108) OPTION_ANI_AP_BSSID',
            value: 108,
        },
        {
            label: '(109) OPTION_ANI_OPERATOR_ID',
            value: 109,
        },
        {
            label: '(110) OPTION_ANI_OPERATOR_REALM',
            value: 110,
        },
        {
            label: '(111) OPTION_S46_PRIORITY',
            value: 111,
        },
        {
            label: '(112) OPTION_MUD_URL_V6',
            value: 112,
        },
        {
            label: '(113) OPTION_V6_PREFIX64',
            value: 113,
        },
        {
            label: '(114) OPTION_F_BINDING_STATUS',
            value: 114,
        },
        {
            label: '(115) OPTION_F_CONNECT_FLAGS',
            value: 115,
        },
        {
            label: '(116) OPTION_F_DNS_REMOVAL_INFO',
            value: 116,
        },
        {
            label: '(117) OPTION_F_DNS_HOST_NAME',
            value: 117,
        },
        {
            label: '(118) OPTION_F_DNS_ZONE_NAME',
            value: 118,
        },
        {
            label: '(119) OPTION_F_DNS_FLAGS',
            value: 119,
        },
        {
            label: '(120) OPTION_F_EXPIRATION_TIME',
            value: 120,
        },
        {
            label: '(121) OPTION_F_MAX_UNACKED_BNDUPD',
            value: 121,
        },
        {
            label: '(122) OPTION_F_MCLT',
            value: 122,
        },
        {
            label: '(123) OPTION_F_PARTNER_LIFETIME',
            value: 123,
        },
        {
            label: '(124) OPTION_F_PARTNER_LIFETIME_SENT',
            value: 124,
        },
        {
            label: '(125) OPTION_F_PARTNER_DOWN_TIME',
            value: 125,
        },
        {
            label: '(126) OPTION_F_PARTNER_RAW_CLT_TIME',
            value: 126,
        },
        {
            label: '(127) OPTION_F_PROTOCOL_VERSION',
            value: 127,
        },
        {
            label: '(128) OPTION_F_KEEPALIVE_TIME',
            value: 128,
        },
        {
            label: '(129) OPTION_F_RECONFIGURE_DATA',
            value: 129,
        },
        {
            label: '(130) OPTION_F_RELATIONSHIP_NAME',
            value: 130,
        },
        {
            label: '(131) OPTION_F_SERVER_FLAGS',
            value: 131,
        },
        {
            label: '(132) OPTION_F_SERVER_STATE',
            value: 132,
        },
        {
            label: '(133) OPTION_F_START_TIME_OF_STATE',
            value: 133,
        },
        {
            label: '(134) OPTION_F_STATE_EXPIRATION_TIME',
            value: 134,
        },
        {
            label: '(135) OPTION_RELAY_PORT',
            value: 135,
        },
        {
            label: '(136) OPTION_V6_SZTP_REDIRECT',
            value: 136,
        },
        {
            label: '(137) OPTION_S46_BIND_IPV6_PREFIX',
            value: 137,
        },
        {
            label: '(138) OPTION_IA_LL',
            value: 138,
        },
        {
            label: '(139) OPTION_LLADDR',
            value: 139,
        },
        {
            label: '(140) OPTION_SLAP_QUAD',
            value: 140,
        },
        {
            label: '(141) OPTION_V6_DOTS_RI',
            value: 141,
        },
        {
            label: '(142) OPTION_V6_DOTS_ADDRESS',
            value: 142,
        },*/
        {
            label: '(143) OPTION-IPv6_Address-ANDSF',
            value: 143,
        },
    ]

    /**
     * Indexes the standard DHCPv6 options by option code for faster lookup.
     */
    private _dhcpv6OptionsByCode: Map<number, DhcpOptionListItem>

    /**
     * Constructor.
     *
     * Creates indexes of the options by the option codes.
     */
    constructor() {
        this._dhcpv4OptionsByCode = new Map(this._dhcpv4Options.map((o) => [o.value, o]))
        this._dhcpv6OptionsByCode = new Map(this._dhcpv6Options.map((o) => [o.value, o]))
    }

    /**
     * Returns configurable standard DHCPv4 options.
     *
     * Returned list can be used to initialize dropdown list of options in a form.
     */
    getStandardDhcpv4Options(): DhcpOptionListItem[] {
        return this._dhcpv4Options
    }

    /**
     * Returns configurable standard DHCPv6 options.
     *
     * Returned list can be used to initialize dropdown list of options in a form.
     */
    getStandardDhcpv6Options(): DhcpOptionListItem[] {
        return this._dhcpv6Options
    }

    /**
     * Finds a specific DHCPv4 option by option code.
     *
     * @param code option code.
     * @returns option description or null if it is not found.
     */
    findStandardDhcpv4Option(code: number): DhcpOptionListItem | null {
        return this._dhcpv4OptionsByCode.get(code)
    }

    /**
     * Finds a specific DHCPv6 option by option code.
     *
     * @param code option code.
     * @returns option description or null if it is not found.
     */
    findStandardDhcpv6Option(code: number): DhcpOptionListItem | null {
        return this._dhcpv6OptionsByCode.get(code)
    }

    /**
     * Finds a standard DHCPv4 option definition by the code and space.
     *
     * @param code option code.
     * @param space option space.
     * @returns DHCPv4 option definition or null, if not found.
     */
    findStandardDhcpv4OptionDef(code: number, space: string | null): DhcpOptionDef | null {
        return stdDhcpv4OptionDefs.find((def) => def.code === code && def.space === (space ?? 'dhcp4'))
    }

    /**
     * Finds a standard DHCPv6 option definition by the code and space.
     *
     * @param code option code.
     * @param space option space.
     * @returns DHCPv6 option definition or null, if not found.
     */
    findStandardDhcpv6OptionDef(code: number, space: string | null): DhcpOptionDef | null {
        return stdDhcpv6OptionDefs.find((def) => def.code === code && def.space === (space ?? 'dhcp6'))
    }

    /**
     * Finds all standard DHCPv4 option definitions in option space.
     *
     * If the option space is null, the top-level dhcp4 option space is assumed.
     *
     * @param space option space name.
     * @returns An array of option definitions in the option space.
     */
    findStandardDhcpv4OptionDefsBySpace(space: string | null): DhcpOptionDef[] {
        return stdDhcpv4OptionDefs.filter((def) => def.space === (space ?? 'dhcp4'))
    }

    /**
     * Finds all standard DHCPv6 option definitions in option space.
     *
     * If the option space is null, the top-level dhcp6 option space is assumed.
     *
     * @param space option space name.
     * @returns An array of option definitions in the option space.
     */
    findStandardDhcpv6OptionDefsBySpace(space: string | null): DhcpOptionDef[] {
        return stdDhcpv6OptionDefs.filter((def) => def.space === (space ?? 'dhcp6'))
    }
}
