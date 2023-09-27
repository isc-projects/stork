package restservice

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	keaconfig "isc.org/stork/appcfg/kea"
	"isc.org/stork/server/apps/kea"
	"isc.org/stork/server/config"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"

	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
)

// Creates a REST API representation of a subnet from a database model.
func (r *RestAPI) convertSubnetToRestAPI(sn *dbmodel.Subnet) *models.Subnet {
	subnet := &models.Subnet{
		ID:               sn.ID,
		Subnet:           sn.Prefix,
		ClientClass:      sn.ClientClass,
		AddrUtilization:  float64(sn.AddrUtilization) / 10,
		PdUtilization:    float64(sn.PdUtilization) / 10,
		Stats:            sn.Stats,
		StatsCollectedAt: convertToOptionalDatetime(sn.StatsCollectedAt),
	}

	if sn.SharedNetwork != nil {
		subnet.SharedNetworkID = sn.SharedNetwork.ID
		subnet.SharedNetwork = sn.SharedNetwork.Name
	}

	for _, lsn := range sn.LocalSubnets {
		localSubnet := &models.LocalSubnet{
			AppID:            lsn.Daemon.App.ID,
			DaemonID:         lsn.Daemon.ID,
			AppName:          lsn.Daemon.App.Name,
			ID:               lsn.LocalSubnetID,
			MachineAddress:   lsn.Daemon.App.Machine.Address,
			MachineHostname:  lsn.Daemon.App.Machine.State.Hostname,
			Stats:            lsn.Stats,
			StatsCollectedAt: convertToOptionalDatetime(lsn.StatsCollectedAt),
		}
		for _, poolDetails := range lsn.AddressPools {
			pool := &models.Pool{
				Pool: storkutil.Ptr(poolDetails.LowerBound + "-" + poolDetails.UpperBound),
			}
			if poolDetails.KeaParameters != nil {
				pool.KeaConfigPoolParameters = &models.KeaConfigPoolParameters{
					KeaConfigClientClassParameters: models.KeaConfigClientClassParameters{
						ClientClass:          storkutil.NullifyEmptyString(poolDetails.KeaParameters.ClientClass),
						RequireClientClasses: poolDetails.KeaParameters.RequireClientClasses,
					},
				}
				// DHCP options.
				pool.KeaConfigPoolParameters.OptionsHash = poolDetails.DHCPOptionSetHash
				pool.KeaConfigPoolParameters.Options = r.unflattenDHCPOptions(poolDetails.DHCPOptionSet, "", 0)
			}

			localSubnet.Pools = append(localSubnet.Pools, pool)
		}

		for _, prefixPoolDetails := range lsn.PrefixPools {
			prefix := prefixPoolDetails.Prefix
			delegatedLength := int64(prefixPoolDetails.DelegatedLen)
			pool := &models.DelegatedPrefixPool{
				Prefix:          &prefix,
				DelegatedLength: &delegatedLength,
				ExcludedPrefix:  prefixPoolDetails.ExcludedPrefix,
			}
			localSubnet.PrefixDelegationPools = append(localSubnet.PrefixDelegationPools, pool)
			if prefixPoolDetails.KeaParameters != nil {
				pool.KeaConfigPoolParameters = &models.KeaConfigPoolParameters{
					KeaConfigClientClassParameters: models.KeaConfigClientClassParameters{
						ClientClass:          storkutil.NullifyEmptyString(prefixPoolDetails.KeaParameters.ClientClass),
						RequireClientClasses: prefixPoolDetails.KeaParameters.RequireClientClasses,
					},
				}
				// DHCP options.
				pool.KeaConfigPoolParameters.OptionsHash = prefixPoolDetails.DHCPOptionSetHash
				pool.KeaConfigPoolParameters.Options = r.unflattenDHCPOptions(prefixPoolDetails.DHCPOptionSet, "", 0)
			}
		}

		// Subnet level Kea DHCP parameters.
		if lsn.KeaParameters != nil {
			keaParameters := lsn.KeaParameters
			if localSubnet.KeaConfigSubnetParameters == nil {
				localSubnet.KeaConfigSubnetParameters = &models.KeaConfigSubnetParameters{}
			}
			localSubnet.KeaConfigSubnetParameters.SubnetLevelParameters = &models.KeaConfigSubnetDerivedParameters{
				KeaConfigCacheParameters: models.KeaConfigCacheParameters{
					CacheThreshold: keaParameters.CacheThreshold,
					CacheMaxAge:    keaParameters.CacheMaxAge,
				},
				KeaConfigClientClassParameters: models.KeaConfigClientClassParameters{
					ClientClass:          storkutil.NullifyEmptyString(keaParameters.ClientClass),
					RequireClientClasses: keaParameters.RequireClientClasses,
				},
				KeaConfigDdnsParameters: models.KeaConfigDdnsParameters{
					DdnsGeneratedPrefix:       storkutil.NullifyEmptyString(keaParameters.DDNSGeneratedPrefix),
					DdnsOverrideClientUpdate:  keaParameters.DDNSOverrideClientUpdate,
					DdnsOverrideNoUpdate:      keaParameters.DDNSOverrideNoUpdate,
					DdnsQualifyingSuffix:      storkutil.NullifyEmptyString(keaParameters.DDNSQualifyingSuffix),
					DdnsReplaceClientName:     storkutil.NullifyEmptyString(keaParameters.DDNSReplaceClientName),
					DdnsSendUpdates:           keaParameters.DDNSSendUpdates,
					DdnsUpdateOnRenew:         keaParameters.DDNSUpdateOnRenew,
					DdnsUseConflictResolution: keaParameters.DDNSUseConflictResolution,
				},
				KeaConfigFourOverSixParameters: models.KeaConfigFourOverSixParameters{
					FourOverSixInterface:   storkutil.NullifyEmptyString(keaParameters.FourOverSixInterface),
					FourOverSixInterfaceID: storkutil.NullifyEmptyString(keaParameters.FourOverSixInterfaceID),
					FourOverSixSubnet:      storkutil.NullifyEmptyString(keaParameters.FourOverSixSubnet),
				},
				KeaConfigHostnameCharParameters: models.KeaConfigHostnameCharParameters{
					HostnameCharReplacement: storkutil.NullifyEmptyString(keaParameters.HostnameCharReplacement),
					HostnameCharSet:         storkutil.NullifyEmptyString(keaParameters.HostnameCharSet),
				},
				KeaConfigPreferredLifetimeParameters: models.KeaConfigPreferredLifetimeParameters{
					MaxPreferredLifetime: keaParameters.MaxPreferredLifetime,
					MinPreferredLifetime: keaParameters.MinPreferredLifetime,
					PreferredLifetime:    keaParameters.PreferredLifetime,
				},
				KeaConfigReservationParameters: models.KeaConfigReservationParameters{
					ReservationMode:       storkutil.NullifyEmptyString(keaParameters.ReservationMode),
					ReservationsGlobal:    keaParameters.ReservationsGlobal,
					ReservationsInSubnet:  keaParameters.ReservationsInSubnet,
					ReservationsOutOfPool: keaParameters.ReservationsOutOfPool,
				},
				KeaConfigTimerParameters: models.KeaConfigTimerParameters{
					CalculateTeeTimes: keaParameters.CalculateTeeTimes,
					RebindTimer:       keaParameters.RebindTimer,
					RenewTimer:        keaParameters.RenewTimer,
					T1Percent:         keaParameters.T1Percent,
					T2Percent:         keaParameters.T2Percent,
				},
				KeaConfigValidLifetimeParameters: models.KeaConfigValidLifetimeParameters{
					MaxValidLifetime: keaParameters.MaxValidLifetime,
					MinValidLifetime: keaParameters.MinValidLifetime,
					ValidLifetime:    keaParameters.ValidLifetime,
				},
				KeaConfigAssortedSubnetParameters: models.KeaConfigAssortedSubnetParameters{
					Allocator:         storkutil.NullifyEmptyString(keaParameters.Allocator),
					Authoritative:     keaParameters.Authoritative,
					BootFileName:      storkutil.NullifyEmptyString(keaParameters.BootFileName),
					Interface:         storkutil.NullifyEmptyString(keaParameters.Interface),
					InterfaceID:       storkutil.NullifyEmptyString(keaParameters.InterfaceID),
					MatchClientID:     keaParameters.MatchClientID,
					NextServer:        storkutil.NullifyEmptyString(keaParameters.NextServer),
					PdAllocator:       storkutil.NullifyEmptyString(keaParameters.PDAllocator),
					RapidCommit:       keaParameters.RapidCommit,
					ServerHostname:    storkutil.NullifyEmptyString(keaParameters.ServerHostname),
					StoreExtendedInfo: keaParameters.StoreExtendedInfo,
				},
			}
			if keaParameters.Relay != nil {
				localSubnet.KeaConfigSubnetParameters.SubnetLevelParameters.Relay = &models.KeaConfigAssortedSubnetParametersRelay{
					IPAddresses: keaParameters.Relay.IPAddresses,
				}
			}
			localSubnet.KeaConfigSubnetParameters.SubnetLevelParameters.OptionsHash = lsn.DHCPOptionSetHash
			localSubnet.KeaConfigSubnetParameters.SubnetLevelParameters.Options = r.unflattenDHCPOptions(lsn.DHCPOptionSet, "", 0)
		}
		// Shared network level Kea DHCP parameters.
		if sn.SharedNetwork != nil {
			keaParameters := sn.SharedNetwork.GetKeaParameters(lsn.DaemonID)
			if keaParameters != nil {
				if localSubnet.KeaConfigSubnetParameters == nil {
					localSubnet.KeaConfigSubnetParameters = &models.KeaConfigSubnetParameters{}
				}
				localSubnet.KeaConfigSubnetParameters.SharedNetworkLevelParameters = &models.KeaConfigSubnetDerivedParameters{
					KeaConfigCacheParameters: models.KeaConfigCacheParameters{
						CacheThreshold: keaParameters.CacheThreshold,
						CacheMaxAge:    keaParameters.CacheMaxAge,
					},
					KeaConfigClientClassParameters: models.KeaConfigClientClassParameters{
						ClientClass:          storkutil.NullifyEmptyString(keaParameters.ClientClass),
						RequireClientClasses: keaParameters.RequireClientClasses,
					},
					KeaConfigDdnsParameters: models.KeaConfigDdnsParameters{
						DdnsGeneratedPrefix:       storkutil.NullifyEmptyString(keaParameters.DDNSGeneratedPrefix),
						DdnsOverrideClientUpdate:  keaParameters.DDNSOverrideClientUpdate,
						DdnsOverrideNoUpdate:      keaParameters.DDNSOverrideNoUpdate,
						DdnsQualifyingSuffix:      storkutil.NullifyEmptyString(keaParameters.DDNSQualifyingSuffix),
						DdnsReplaceClientName:     storkutil.NullifyEmptyString(keaParameters.DDNSReplaceClientName),
						DdnsSendUpdates:           keaParameters.DDNSSendUpdates,
						DdnsUpdateOnRenew:         keaParameters.DDNSUpdateOnRenew,
						DdnsUseConflictResolution: keaParameters.DDNSUseConflictResolution,
					},
					KeaConfigHostnameCharParameters: models.KeaConfigHostnameCharParameters{
						HostnameCharReplacement: storkutil.NullifyEmptyString(keaParameters.HostnameCharReplacement),
						HostnameCharSet:         storkutil.NullifyEmptyString(keaParameters.HostnameCharSet),
					},
					KeaConfigPreferredLifetimeParameters: models.KeaConfigPreferredLifetimeParameters{
						MaxPreferredLifetime: keaParameters.MaxPreferredLifetime,
						MinPreferredLifetime: keaParameters.MinPreferredLifetime,
						PreferredLifetime:    keaParameters.PreferredLifetime,
					},
					KeaConfigReservationParameters: models.KeaConfigReservationParameters{
						ReservationMode:       storkutil.NullifyEmptyString(keaParameters.ReservationMode),
						ReservationsGlobal:    keaParameters.ReservationsGlobal,
						ReservationsInSubnet:  keaParameters.ReservationsInSubnet,
						ReservationsOutOfPool: keaParameters.ReservationsOutOfPool,
					},
					KeaConfigTimerParameters: models.KeaConfigTimerParameters{
						CalculateTeeTimes: keaParameters.CalculateTeeTimes,
						RebindTimer:       keaParameters.RebindTimer,
						RenewTimer:        keaParameters.RenewTimer,
						T1Percent:         keaParameters.T1Percent,
						T2Percent:         keaParameters.T2Percent,
					},
					KeaConfigValidLifetimeParameters: models.KeaConfigValidLifetimeParameters{
						MaxValidLifetime: keaParameters.MaxValidLifetime,
						MinValidLifetime: keaParameters.MinValidLifetime,
						ValidLifetime:    keaParameters.ValidLifetime,
					},
					KeaConfigAssortedSubnetParameters: models.KeaConfigAssortedSubnetParameters{
						Allocator:         storkutil.NullifyEmptyString(keaParameters.Allocator),
						Authoritative:     keaParameters.Authoritative,
						BootFileName:      storkutil.NullifyEmptyString(keaParameters.BootFileName),
						Interface:         storkutil.NullifyEmptyString(keaParameters.Interface),
						InterfaceID:       storkutil.NullifyEmptyString(keaParameters.InterfaceID),
						MatchClientID:     keaParameters.MatchClientID,
						NextServer:        storkutil.NullifyEmptyString(keaParameters.NextServer),
						PdAllocator:       storkutil.NullifyEmptyString(keaParameters.PDAllocator),
						RapidCommit:       keaParameters.RapidCommit,
						ServerHostname:    storkutil.NullifyEmptyString(keaParameters.ServerHostname),
						StoreExtendedInfo: keaParameters.StoreExtendedInfo,
					},
				}
				if keaParameters.Relay != nil {
					localSubnet.KeaConfigSubnetParameters.SharedNetworkLevelParameters.Relay = &models.KeaConfigAssortedSubnetParametersRelay{
						IPAddresses: keaParameters.Relay.IPAddresses,
					}
				}
				if localSharedNetwork := sn.SharedNetwork.GetLocalSharedNetwork(lsn.DaemonID); localSharedNetwork != nil {
					localSubnet.KeaConfigSubnetParameters.SharedNetworkLevelParameters.OptionsHash = localSharedNetwork.DHCPOptionSetHash
					localSubnet.KeaConfigSubnetParameters.SharedNetworkLevelParameters.Options = r.unflattenDHCPOptions(localSharedNetwork.DHCPOptionSet, "", 0)
				}
			}
		}

		// Global configuration parameters.
		if lsn.Daemon != nil && lsn.Daemon.KeaDaemon != nil && lsn.Daemon.KeaDaemon.Config != nil &&
			(lsn.Daemon.KeaDaemon.Config.IsDHCPv4() || lsn.Daemon.KeaDaemon.Config.IsDHCPv6()) {
			cfg := lsn.Daemon.KeaDaemon.Config
			if localSubnet.KeaConfigSubnetParameters == nil {
				localSubnet.KeaConfigSubnetParameters = &models.KeaConfigSubnetParameters{}
			}
			localSubnet.KeaConfigSubnetParameters.GlobalParameters = &models.KeaConfigSubnetDerivedParameters{
				KeaConfigCacheParameters: models.KeaConfigCacheParameters{
					CacheThreshold: cfg.GetCacheParameters().CacheThreshold,
					CacheMaxAge:    cfg.GetCacheParameters().CacheMaxAge,
				},
				KeaConfigDdnsParameters: models.KeaConfigDdnsParameters{
					DdnsGeneratedPrefix:       storkutil.NullifyEmptyString(cfg.GetDDNSParameters().DDNSGeneratedPrefix),
					DdnsOverrideClientUpdate:  cfg.GetDDNSParameters().DDNSOverrideClientUpdate,
					DdnsOverrideNoUpdate:      cfg.GetDDNSParameters().DDNSOverrideNoUpdate,
					DdnsQualifyingSuffix:      storkutil.NullifyEmptyString(cfg.GetDDNSParameters().DDNSQualifyingSuffix),
					DdnsReplaceClientName:     storkutil.NullifyEmptyString(cfg.GetDDNSParameters().DDNSReplaceClientName),
					DdnsSendUpdates:           cfg.GetDDNSParameters().DDNSSendUpdates,
					DdnsUpdateOnRenew:         cfg.GetDDNSParameters().DDNSUpdateOnRenew,
					DdnsUseConflictResolution: cfg.GetDDNSParameters().DDNSUseConflictResolution,
				},
				KeaConfigHostnameCharParameters: models.KeaConfigHostnameCharParameters{
					HostnameCharReplacement: storkutil.NullifyEmptyString(cfg.GetHostnameCharParameters().HostnameCharReplacement),
					HostnameCharSet:         storkutil.NullifyEmptyString(cfg.GetHostnameCharParameters().HostnameCharSet),
				},
				KeaConfigPreferredLifetimeParameters: models.KeaConfigPreferredLifetimeParameters{
					MaxPreferredLifetime: cfg.GetPreferredLifetimeParameters().MaxPreferredLifetime,
					MinPreferredLifetime: cfg.GetPreferredLifetimeParameters().MinPreferredLifetime,
					PreferredLifetime:    cfg.GetPreferredLifetimeParameters().PreferredLifetime,
				},
				KeaConfigReservationParameters: models.KeaConfigReservationParameters{
					ReservationMode:       storkutil.NullifyEmptyString(cfg.GetGlobalReservationParameters().ReservationMode),
					ReservationsGlobal:    cfg.GetGlobalReservationParameters().ReservationsGlobal,
					ReservationsInSubnet:  cfg.GetGlobalReservationParameters().ReservationsInSubnet,
					ReservationsOutOfPool: cfg.GetGlobalReservationParameters().ReservationsOutOfPool,
				},
				KeaConfigTimerParameters: models.KeaConfigTimerParameters{
					CalculateTeeTimes: cfg.GetTimerParameters().CalculateTeeTimes,
					RebindTimer:       cfg.GetTimerParameters().RebindTimer,
					RenewTimer:        cfg.GetTimerParameters().RenewTimer,
					T1Percent:         cfg.GetTimerParameters().T1Percent,
					T2Percent:         cfg.GetTimerParameters().T2Percent,
				},
				KeaConfigValidLifetimeParameters: models.KeaConfigValidLifetimeParameters{
					MaxValidLifetime: cfg.GetValidLifetimeParameters().MaxValidLifetime,
					MinValidLifetime: cfg.GetValidLifetimeParameters().MinValidLifetime,
					ValidLifetime:    cfg.GetValidLifetimeParameters().ValidLifetime,
				},
				KeaConfigAssortedSubnetParameters: models.KeaConfigAssortedSubnetParameters{
					Allocator:         storkutil.NullifyEmptyString(cfg.GetAllocator()),
					Authoritative:     cfg.GetAuthoritative(),
					BootFileName:      storkutil.NullifyEmptyString(cfg.GetBootFileName()),
					MatchClientID:     cfg.GetMatchClientID(),
					NextServer:        storkutil.NullifyEmptyString(cfg.GetNextServer()),
					PdAllocator:       storkutil.NullifyEmptyString(cfg.GetPDAllocator()),
					RapidCommit:       cfg.GetRapidCommit(),
					ServerHostname:    storkutil.NullifyEmptyString(cfg.GetServerHostname()),
					StoreExtendedInfo: cfg.GetStoreExtendedInfo(),
				},
			}
			var convertedOptions []dbmodel.DHCPOption
			for _, option := range cfg.GetDHCPOptions() {
				convertedOption, err := dbmodel.NewDHCPOptionFromKea(option, storkutil.IPType(sn.GetFamily()), r.DHCPOptionDefinitionLookup)
				if err != nil {
					continue
				}
				convertedOptions = append(convertedOptions, *convertedOption)
			}
			localSubnet.KeaConfigSubnetParameters.GlobalParameters.OptionsHash = storkutil.Fnv128(convertedOptions)
			localSubnet.KeaConfigSubnetParameters.GlobalParameters.Options = r.unflattenDHCPOptions(convertedOptions, "", 0)
		}
		subnet.LocalSubnets = append(subnet.LocalSubnets, localSubnet)
	}
	return subnet
}

// Convert subnet from the format used in REST API to a database subnet representation.
// It is used when Stork user modifies or creates new subnet. Thus, it doesn't populate
// subnet statistics as it is not specified by Stork user. It is pulled from the Kea
// servers periodically.
func (r *RestAPI) convertSubnetFromRestAPI(restSubnet *models.Subnet) (*dbmodel.Subnet, error) {
	subnet := &dbmodel.Subnet{
		ID:              restSubnet.ID,
		Prefix:          restSubnet.Subnet,
		ClientClass:     restSubnet.ClientClass,
		SharedNetworkID: restSubnet.SharedNetworkID,
	}
	// Convert local subnet containing associations of the subnet with daemons.
	for _, ls := range restSubnet.LocalSubnets {
		localSubnet := &dbmodel.LocalSubnet{
			LocalSubnetID: ls.ID,
			DaemonID:      ls.DaemonID,
		}
		for _, poolDetails := range ls.Pools {
			pool, err := dbmodel.NewAddressPoolFromRange(*poolDetails.Pool)
			if err != nil {
				return nil, err
			}
			if poolDetails.KeaConfigPoolParameters != nil {
				pool.KeaParameters = &keaconfig.PoolParameters{
					ClientClassParameters: keaconfig.ClientClassParameters{
						ClientClass:          storkutil.NullifyEmptyString(poolDetails.KeaConfigPoolParameters.ClientClass),
						RequireClientClasses: poolDetails.KeaConfigPoolParameters.RequireClientClasses,
					},
				}
				// DHCP options.
				pool.DHCPOptionSet, err = r.flattenDHCPOptions("", poolDetails.KeaConfigPoolParameters.Options, 0)
				if err != nil {
					return nil, err
				}
				if len(pool.DHCPOptionSet) > 0 {
					pool.DHCPOptionSetHash = storkutil.Fnv128(pool.DHCPOptionSet)
				}
			}
			localSubnet.AddressPools = append(localSubnet.AddressPools, *pool)
		}

		for _, prefixPoolDetails := range ls.PrefixDelegationPools {
			pool, err := dbmodel.NewPrefixPool(*prefixPoolDetails.Prefix, int(*prefixPoolDetails.DelegatedLength), prefixPoolDetails.ExcludedPrefix)
			if err != nil {
				return nil, err
			}
			if prefixPoolDetails.KeaConfigPoolParameters != nil {
				pool.KeaParameters = &keaconfig.PoolParameters{
					ClientClassParameters: keaconfig.ClientClassParameters{
						ClientClass:          storkutil.NullifyEmptyString(prefixPoolDetails.KeaConfigPoolParameters.ClientClass),
						RequireClientClasses: prefixPoolDetails.KeaConfigPoolParameters.RequireClientClasses,
					},
				}
				// DHCP options.
				pool.DHCPOptionSet, err = r.flattenDHCPOptions("", prefixPoolDetails.KeaConfigPoolParameters.Options, 0)
				if err != nil {
					return nil, err
				}
				if len(pool.DHCPOptionSet) > 0 {
					pool.DHCPOptionSetHash = storkutil.Fnv128(pool.DHCPOptionSet)
				}
			}
			localSubnet.PrefixPools = append(localSubnet.PrefixPools, *pool)
		}
		var err error
		if ls.KeaConfigSubnetParameters != nil && ls.KeaConfigSubnetParameters.SubnetLevelParameters != nil {
			keaParameters := ls.KeaConfigSubnetParameters.SubnetLevelParameters
			localSubnet.KeaParameters = &keaconfig.SubnetParameters{
				CacheParameters: keaconfig.CacheParameters{
					CacheThreshold: keaParameters.CacheThreshold,
					CacheMaxAge:    keaParameters.CacheMaxAge,
				},
				ClientClassParameters: keaconfig.ClientClassParameters{
					ClientClass:          storkutil.NullifyEmptyString(keaParameters.ClientClass),
					RequireClientClasses: keaParameters.RequireClientClasses,
				},
				DDNSParameters: keaconfig.DDNSParameters{
					DDNSGeneratedPrefix:       storkutil.NullifyEmptyString(keaParameters.DdnsGeneratedPrefix),
					DDNSOverrideClientUpdate:  keaParameters.DdnsOverrideClientUpdate,
					DDNSOverrideNoUpdate:      keaParameters.DdnsOverrideNoUpdate,
					DDNSQualifyingSuffix:      storkutil.NullifyEmptyString(keaParameters.DdnsQualifyingSuffix),
					DDNSReplaceClientName:     storkutil.NullifyEmptyString(keaParameters.DdnsReplaceClientName),
					DDNSSendUpdates:           keaParameters.DdnsSendUpdates,
					DDNSUpdateOnRenew:         keaParameters.DdnsUpdateOnRenew,
					DDNSUseConflictResolution: keaParameters.DdnsUseConflictResolution,
				},
				FourOverSixParameters: keaconfig.FourOverSixParameters{
					FourOverSixInterface:   storkutil.NullifyEmptyString(keaParameters.FourOverSixInterface),
					FourOverSixInterfaceID: storkutil.NullifyEmptyString(keaParameters.FourOverSixInterfaceID),
					FourOverSixSubnet:      storkutil.NullifyEmptyString(keaParameters.FourOverSixSubnet),
				},
				HostnameCharParameters: keaconfig.HostnameCharParameters{
					HostnameCharReplacement: storkutil.NullifyEmptyString(keaParameters.HostnameCharReplacement),
					HostnameCharSet:         storkutil.NullifyEmptyString(keaParameters.HostnameCharSet),
				},
				PreferredLifetimeParameters: keaconfig.PreferredLifetimeParameters{
					MaxPreferredLifetime: keaParameters.MaxPreferredLifetime,
					MinPreferredLifetime: keaParameters.MinPreferredLifetime,
					PreferredLifetime:    keaParameters.PreferredLifetime,
				},
				ReservationParameters: keaconfig.ReservationParameters{
					ReservationMode:       storkutil.NullifyEmptyString(keaParameters.ReservationMode),
					ReservationsGlobal:    keaParameters.ReservationsGlobal,
					ReservationsInSubnet:  keaParameters.ReservationsInSubnet,
					ReservationsOutOfPool: keaParameters.ReservationsOutOfPool,
				},
				TimerParameters: keaconfig.TimerParameters{
					CalculateTeeTimes: keaParameters.CalculateTeeTimes,
					RebindTimer:       keaParameters.RebindTimer,
					RenewTimer:        keaParameters.RenewTimer,
					T1Percent:         keaParameters.T1Percent,
					T2Percent:         keaParameters.T2Percent,
				},
				ValidLifetimeParameters: keaconfig.ValidLifetimeParameters{
					MaxValidLifetime: keaParameters.MaxValidLifetime,
					MinValidLifetime: keaParameters.MinValidLifetime,
					ValidLifetime:    keaParameters.ValidLifetime,
				},
				Allocator:         storkutil.NullifyEmptyString(keaParameters.Allocator),
				Authoritative:     keaParameters.Authoritative,
				BootFileName:      storkutil.NullifyEmptyString(keaParameters.BootFileName),
				Interface:         storkutil.NullifyEmptyString(keaParameters.Interface),
				InterfaceID:       storkutil.NullifyEmptyString(keaParameters.InterfaceID),
				MatchClientID:     keaParameters.MatchClientID,
				NextServer:        storkutil.NullifyEmptyString(keaParameters.NextServer),
				PDAllocator:       storkutil.NullifyEmptyString(keaParameters.PdAllocator),
				RapidCommit:       keaParameters.RapidCommit,
				ServerHostname:    storkutil.NullifyEmptyString(keaParameters.ServerHostname),
				StoreExtendedInfo: keaParameters.StoreExtendedInfo,
			}
			if keaParameters.Relay != nil {
				localSubnet.KeaParameters.Relay = &keaconfig.Relay{
					IPAddresses: keaParameters.Relay.IPAddresses,
				}
			}
			// DHCP options.
			localSubnet.DHCPOptionSet, err = r.flattenDHCPOptions("", ls.KeaConfigSubnetParameters.SubnetLevelParameters.Options, 0)
			if err != nil {
				return nil, err
			}
			if len(localSubnet.DHCPOptionSet) > 0 {
				localSubnet.DHCPOptionSetHash = storkutil.Fnv128(localSubnet.DHCPOptionSet)
			}
		}
		subnet.SetLocalSubnet(localSubnet)
	}
	return subnet, nil
}

func (r *RestAPI) getSubnets(offset, limit int64, filters *dbmodel.SubnetsByPageFilters, sortField string, sortDir dbmodel.SortDirEnum) (*models.Subnets, error) {
	// get subnets from db
	dbSubnets, total, err := dbmodel.GetSubnetsByPage(r.DB, offset, limit, filters, sortField, sortDir)
	if err != nil {
		return nil, err
	}

	// prepare response
	subnets := &models.Subnets{
		Total: total,
	}

	// go through subnets from db and change their format to ReST one
	for _, snTmp := range dbSubnets {
		sn := snTmp
		subnet := r.convertSubnetToRestAPI(&sn)
		subnets.Items = append(subnets.Items, subnet)
	}

	return subnets, nil
}

// Get list of DHCP subnets. The list can be filtered by app ID, DHCP version and text.
func (r *RestAPI) GetSubnets(ctx context.Context, params dhcp.GetSubnetsParams) middleware.Responder {
	var start int64
	if params.Start != nil {
		start = *params.Start
	}

	var limit int64 = 10
	if params.Limit != nil {
		limit = *params.Limit
	}

	// get subnets from db
	filters := &dbmodel.SubnetsByPageFilters{
		AppID:         params.AppID,
		Family:        params.DhcpVersion,
		Text:          params.Text,
		LocalSubnetID: params.LocalSubnetID,
	}

	subnets, err := r.getSubnets(start, limit, filters, "", dbmodel.SortDirAsc)
	if err != nil {
		msg := "Cannot get subnets from db"
		log.Error(err)
		rsp := dhcp.NewGetSubnetsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	rsp := dhcp.NewGetSubnetsOK().WithPayload(subnets)
	return rsp
}

// Returns the detailed subnet information including the subnet, shared network and
// global DHCP configuration parameters. The returned information is sufficient to
// open a form for editing the subnet.
func (r *RestAPI) GetSubnet(ctx context.Context, params dhcp.GetSubnetParams) middleware.Responder {
	dbSubnet, err := dbmodel.GetSubnet(r.DB, params.ID)
	if err != nil {
		// Error while communicating with the database.
		msg := fmt.Sprintf("Problem fetching subnet with ID %d from db", params.ID)
		log.WithError(err).Error(msg)
		rsp := dhcp.NewGetSubnetDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	if dbSubnet == nil {
		// Subnet not found.
		msg := fmt.Sprintf("Cannot find subnet with ID %d", params.ID)
		log.Error(msg)
		rsp := dhcp.NewGetSubnetDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	subnet := r.convertSubnetToRestAPI(dbSubnet)
	rsp := dhcp.NewGetSubnetOK().WithPayload(subnet)
	return rsp
}

// Creates a REST API representation of a shared network from a database model.
func (r *RestAPI) sharedNetworkToRestAPI(sn *dbmodel.SharedNetwork) *models.SharedNetwork {
	subnets := []*models.Subnet{}
	// Exclude the subnets that are not attached to any app. This shouldn't
	// be the case but let's be safe.
	for _, snTmp := range sn.Subnets {
		sn := snTmp
		subnet := r.convertSubnetToRestAPI(&sn)
		subnets = append(subnets, subnet)
	}
	// Create shared network.
	sharedNetwork := &models.SharedNetwork{
		ID:               sn.ID,
		Name:             sn.Name,
		Universe:         int64(sn.Family),
		Subnets:          subnets,
		AddrUtilization:  float64(sn.AddrUtilization) / 10,
		PdUtilization:    float64(sn.PdUtilization) / 10,
		Stats:            sn.Stats,
		StatsCollectedAt: convertToOptionalDatetime(sn.StatsCollectedAt),
	}

	for _, lsn := range sn.LocalSharedNetworks {
		localSharedNetwork := &models.LocalSharedNetwork{
			AppID:    lsn.Daemon.App.ID,
			DaemonID: lsn.Daemon.ID,
			AppName:  lsn.Daemon.App.Name,
		}
		keaParameters := lsn.KeaParameters
		if keaParameters != nil {
			if localSharedNetwork.KeaConfigSharedNetworkParameters == nil {
				localSharedNetwork.KeaConfigSharedNetworkParameters = &models.KeaConfigSharedNetworkParameters{}
			}
			localSharedNetwork.KeaConfigSharedNetworkParameters.SharedNetworkLevelParameters = &models.KeaConfigSubnetDerivedParameters{
				KeaConfigCacheParameters: models.KeaConfigCacheParameters{
					CacheThreshold: keaParameters.CacheThreshold,
					CacheMaxAge:    keaParameters.CacheMaxAge,
				},
				KeaConfigClientClassParameters: models.KeaConfigClientClassParameters{
					ClientClass:          storkutil.NullifyEmptyString(keaParameters.ClientClass),
					RequireClientClasses: keaParameters.RequireClientClasses,
				},
				KeaConfigDdnsParameters: models.KeaConfigDdnsParameters{
					DdnsGeneratedPrefix:       storkutil.NullifyEmptyString(keaParameters.DDNSGeneratedPrefix),
					DdnsOverrideClientUpdate:  keaParameters.DDNSOverrideClientUpdate,
					DdnsOverrideNoUpdate:      keaParameters.DDNSOverrideNoUpdate,
					DdnsQualifyingSuffix:      storkutil.NullifyEmptyString(keaParameters.DDNSQualifyingSuffix),
					DdnsReplaceClientName:     storkutil.NullifyEmptyString(keaParameters.DDNSReplaceClientName),
					DdnsSendUpdates:           keaParameters.DDNSSendUpdates,
					DdnsUpdateOnRenew:         keaParameters.DDNSUpdateOnRenew,
					DdnsUseConflictResolution: keaParameters.DDNSUseConflictResolution,
				},
				KeaConfigHostnameCharParameters: models.KeaConfigHostnameCharParameters{
					HostnameCharReplacement: storkutil.NullifyEmptyString(keaParameters.HostnameCharReplacement),
					HostnameCharSet:         storkutil.NullifyEmptyString(keaParameters.HostnameCharSet),
				},
				KeaConfigPreferredLifetimeParameters: models.KeaConfigPreferredLifetimeParameters{
					MaxPreferredLifetime: keaParameters.MaxPreferredLifetime,
					MinPreferredLifetime: keaParameters.MinPreferredLifetime,
					PreferredLifetime:    keaParameters.PreferredLifetime,
				},
				KeaConfigReservationParameters: models.KeaConfigReservationParameters{
					ReservationMode:       storkutil.NullifyEmptyString(keaParameters.ReservationMode),
					ReservationsGlobal:    keaParameters.ReservationsGlobal,
					ReservationsInSubnet:  keaParameters.ReservationsInSubnet,
					ReservationsOutOfPool: keaParameters.ReservationsOutOfPool,
				},
				KeaConfigTimerParameters: models.KeaConfigTimerParameters{
					CalculateTeeTimes: keaParameters.CalculateTeeTimes,
					RebindTimer:       keaParameters.RebindTimer,
					RenewTimer:        keaParameters.RenewTimer,
					T1Percent:         keaParameters.T1Percent,
					T2Percent:         keaParameters.T2Percent,
				},
				KeaConfigValidLifetimeParameters: models.KeaConfigValidLifetimeParameters{
					MaxValidLifetime: keaParameters.MaxValidLifetime,
					MinValidLifetime: keaParameters.MinValidLifetime,
					ValidLifetime:    keaParameters.ValidLifetime,
				},
				KeaConfigAssortedSubnetParameters: models.KeaConfigAssortedSubnetParameters{
					Allocator:         storkutil.NullifyEmptyString(keaParameters.Allocator),
					Authoritative:     keaParameters.Authoritative,
					BootFileName:      storkutil.NullifyEmptyString(keaParameters.BootFileName),
					Interface:         storkutil.NullifyEmptyString(keaParameters.Interface),
					InterfaceID:       storkutil.NullifyEmptyString(keaParameters.InterfaceID),
					MatchClientID:     keaParameters.MatchClientID,
					NextServer:        storkutil.NullifyEmptyString(keaParameters.NextServer),
					PdAllocator:       storkutil.NullifyEmptyString(keaParameters.PDAllocator),
					RapidCommit:       keaParameters.RapidCommit,
					ServerHostname:    storkutil.NullifyEmptyString(keaParameters.ServerHostname),
					StoreExtendedInfo: keaParameters.StoreExtendedInfo,
				},
			}
			if keaParameters.Relay != nil {
				localSharedNetwork.KeaConfigSharedNetworkParameters.SharedNetworkLevelParameters.Relay = &models.KeaConfigAssortedSubnetParametersRelay{
					IPAddresses: keaParameters.Relay.IPAddresses,
				}
			}
			localSharedNetwork.KeaConfigSharedNetworkParameters.SharedNetworkLevelParameters.OptionsHash = lsn.DHCPOptionSetHash
			localSharedNetwork.KeaConfigSharedNetworkParameters.SharedNetworkLevelParameters.Options = r.unflattenDHCPOptions(lsn.DHCPOptionSet, "", 0)
		}

		// Global configuration parameters.
		if lsn.Daemon != nil && lsn.Daemon.KeaDaemon != nil && lsn.Daemon.KeaDaemon.Config != nil &&
			(lsn.Daemon.KeaDaemon.Config.IsDHCPv4() || lsn.Daemon.KeaDaemon.Config.IsDHCPv6()) {
			cfg := lsn.Daemon.KeaDaemon.Config
			if localSharedNetwork.KeaConfigSharedNetworkParameters == nil {
				localSharedNetwork.KeaConfigSharedNetworkParameters = &models.KeaConfigSharedNetworkParameters{}
			}
			localSharedNetwork.KeaConfigSharedNetworkParameters.GlobalParameters = &models.KeaConfigSubnetDerivedParameters{
				KeaConfigCacheParameters: models.KeaConfigCacheParameters{
					CacheThreshold: cfg.GetCacheParameters().CacheThreshold,
					CacheMaxAge:    cfg.GetCacheParameters().CacheMaxAge,
				},
				KeaConfigDdnsParameters: models.KeaConfigDdnsParameters{
					DdnsGeneratedPrefix:       storkutil.NullifyEmptyString(cfg.GetDDNSParameters().DDNSGeneratedPrefix),
					DdnsOverrideClientUpdate:  cfg.GetDDNSParameters().DDNSOverrideClientUpdate,
					DdnsOverrideNoUpdate:      cfg.GetDDNSParameters().DDNSOverrideNoUpdate,
					DdnsQualifyingSuffix:      storkutil.NullifyEmptyString(cfg.GetDDNSParameters().DDNSQualifyingSuffix),
					DdnsReplaceClientName:     storkutil.NullifyEmptyString(cfg.GetDDNSParameters().DDNSReplaceClientName),
					DdnsSendUpdates:           cfg.GetDDNSParameters().DDNSSendUpdates,
					DdnsUpdateOnRenew:         cfg.GetDDNSParameters().DDNSUpdateOnRenew,
					DdnsUseConflictResolution: cfg.GetDDNSParameters().DDNSUseConflictResolution,
				},
				KeaConfigHostnameCharParameters: models.KeaConfigHostnameCharParameters{
					HostnameCharReplacement: storkutil.NullifyEmptyString(cfg.GetHostnameCharParameters().HostnameCharReplacement),
					HostnameCharSet:         storkutil.NullifyEmptyString(cfg.GetHostnameCharParameters().HostnameCharSet),
				},
				KeaConfigPreferredLifetimeParameters: models.KeaConfigPreferredLifetimeParameters{
					MaxPreferredLifetime: cfg.GetPreferredLifetimeParameters().MaxPreferredLifetime,
					MinPreferredLifetime: cfg.GetPreferredLifetimeParameters().MinPreferredLifetime,
					PreferredLifetime:    cfg.GetPreferredLifetimeParameters().PreferredLifetime,
				},
				KeaConfigReservationParameters: models.KeaConfigReservationParameters{
					ReservationMode:       storkutil.NullifyEmptyString(cfg.GetGlobalReservationParameters().ReservationMode),
					ReservationsGlobal:    cfg.GetGlobalReservationParameters().ReservationsGlobal,
					ReservationsInSubnet:  cfg.GetGlobalReservationParameters().ReservationsInSubnet,
					ReservationsOutOfPool: cfg.GetGlobalReservationParameters().ReservationsOutOfPool,
				},
				KeaConfigTimerParameters: models.KeaConfigTimerParameters{
					CalculateTeeTimes: cfg.GetTimerParameters().CalculateTeeTimes,
					RebindTimer:       cfg.GetTimerParameters().RebindTimer,
					RenewTimer:        cfg.GetTimerParameters().RenewTimer,
					T1Percent:         cfg.GetTimerParameters().T1Percent,
					T2Percent:         cfg.GetTimerParameters().T2Percent,
				},
				KeaConfigValidLifetimeParameters: models.KeaConfigValidLifetimeParameters{
					MaxValidLifetime: cfg.GetValidLifetimeParameters().MaxValidLifetime,
					MinValidLifetime: cfg.GetValidLifetimeParameters().MinValidLifetime,
					ValidLifetime:    cfg.GetValidLifetimeParameters().ValidLifetime,
				},
				KeaConfigAssortedSubnetParameters: models.KeaConfigAssortedSubnetParameters{
					Allocator:         storkutil.NullifyEmptyString(cfg.GetAllocator()),
					Authoritative:     cfg.GetAuthoritative(),
					BootFileName:      storkutil.NullifyEmptyString(cfg.GetBootFileName()),
					MatchClientID:     cfg.GetMatchClientID(),
					NextServer:        storkutil.NullifyEmptyString(cfg.GetNextServer()),
					PdAllocator:       storkutil.NullifyEmptyString(cfg.GetPDAllocator()),
					RapidCommit:       cfg.GetRapidCommit(),
					ServerHostname:    storkutil.NullifyEmptyString(cfg.GetServerHostname()),
					StoreExtendedInfo: cfg.GetStoreExtendedInfo(),
				},
			}
			var convertedOptions []dbmodel.DHCPOption
			for _, option := range cfg.GetDHCPOptions() {
				convertedOption, err := dbmodel.NewDHCPOptionFromKea(option, storkutil.IPType(sn.Family), r.DHCPOptionDefinitionLookup)
				if err != nil {
					continue
				}
				convertedOptions = append(convertedOptions, *convertedOption)
			}
			localSharedNetwork.KeaConfigSharedNetworkParameters.GlobalParameters.OptionsHash = storkutil.Fnv128(convertedOptions)
			localSharedNetwork.KeaConfigSharedNetworkParameters.GlobalParameters.Options = r.unflattenDHCPOptions(convertedOptions, "", 0)
		}
		sharedNetwork.LocalSharedNetworks = append(sharedNetwork.LocalSharedNetworks, localSharedNetwork)
	}

	return sharedNetwork
}

func (r *RestAPI) getSharedNetworks(offset, limit, appID, family int64, filterText *string, sortField string, sortDir dbmodel.SortDirEnum) (*models.SharedNetworks, error) {
	// get shared networks from db
	dbSharedNetworks, total, err := dbmodel.GetSharedNetworksByPage(r.DB, offset, limit, appID, family, filterText, sortField, sortDir)
	if err != nil {
		return nil, err
	}

	// prepare response
	sharedNetworks := &models.SharedNetworks{
		Total: total,
	}

	// go through shared networks and their subnets from db and change their format to ReST one
	for i := range dbSharedNetworks {
		if len(dbSharedNetworks[i].Subnets) == 0 || len(dbSharedNetworks[i].Subnets[0].LocalSubnets) == 0 {
			continue
		}
		sharedNetworks.Items = append(sharedNetworks.Items, r.sharedNetworkToRestAPI(&dbSharedNetworks[i]))
	}

	return sharedNetworks, nil
}

// Get list of DHCP shared networks. The list can be filtered by app ID, DHCP version and text.
func (r *RestAPI) GetSharedNetworks(ctx context.Context, params dhcp.GetSharedNetworksParams) middleware.Responder {
	var start int64
	if params.Start != nil {
		start = *params.Start
	}

	var limit int64 = 10
	if params.Limit != nil {
		limit = *params.Limit
	}

	var appID int64
	if params.AppID != nil {
		appID = *params.AppID
	}

	var dhcpVer int64
	if params.DhcpVersion != nil {
		dhcpVer = *params.DhcpVersion
	}

	// get shared networks from db
	sharedNetworks, err := r.getSharedNetworks(start, limit, appID, dhcpVer, params.Text, "", dbmodel.SortDirAsc)
	if err != nil {
		msg := "Cannot get shared network from db"
		log.Error(err)
		rsp := dhcp.NewGetSharedNetworksDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	rsp := dhcp.NewGetSharedNetworksOK().WithPayload(sharedNetworks)
	return rsp
}

// Returns the detailed shared network information including the shared network and
// global DHCP configuration parameters. The returned information is sufficient to
// open a form for editing the shared network.
func (r *RestAPI) GetSharedNetwork(ctx context.Context, params dhcp.GetSharedNetworkParams) middleware.Responder {
	dbSharedNetwork, err := dbmodel.GetSharedNetwork(r.DB, params.ID)
	if err != nil {
		// Error while communicating with the database.
		msg := fmt.Sprintf("Problem fetching shared network with ID %d from db", params.ID)
		log.WithError(err).Error(msg)
		rsp := dhcp.NewGetSharedNetworkDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	if dbSharedNetwork == nil {
		// Subnet not found.
		msg := fmt.Sprintf("Cannot find shared network with ID %d", params.ID)
		log.Error(msg)
		rsp := dhcp.NewGetSharedNetworkDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	sharedNetwork := r.sharedNetworkToRestAPI(dbSharedNetwork)
	rsp := dhcp.NewGetSharedNetworkOK().WithPayload(sharedNetwork)
	return rsp
}

// Common function for executed when creating a new transaction for when the
// subnet is created or updated. It fetches available DHCP daemons. It also
// creates transaction context. If an error occurs, an http error code and
// message are returned.
func (r *RestAPI) commonCreateOrUpdateSubnetBegin(ctx context.Context) ([]*models.KeaDaemon, context.Context, int, string) {
	// A list of Kea DHCP daemons will be needed in the user form,
	// so the user can select which servers send the subnet to.
	daemons, err := dbmodel.GetKeaDHCPDaemons(r.DB)
	if err != nil {
		msg := "Problem with fetching Kea daemons from the database"
		log.Error(err)
		return nil, nil, http.StatusInternalServerError, msg
	}
	// Convert daemons list to REST API format and extract their configured
	// client classes.
	respDaemons := []*models.KeaDaemon{}
	for i := range daemons {
		if daemons[i].KeaDaemon != nil && daemons[i].KeaDaemon.Config != nil {
			// Filter the daemons with subnet_cmds hook library.
			if _, _, exists := daemons[i].KeaDaemon.Config.GetHookLibrary("libdhcp_subnet_cmds"); exists {
				respDaemons = append(respDaemons, keaDaemonToRestAPI(&daemons[i]))
			}
		}
	}

	// If there are no daemons with subnet_cmds hooks library loaded there is no way
	// to add new host reservation. In that case, we don't begin a transaction.
	if len(respDaemons) == 0 {
		msg := "Unable to begin transaction for adding new subnet because there are no Kea servers with subnet_cmds hooks library available"
		log.Error(msg)
		return nil, nil, http.StatusBadRequest, msg
	}
	// Create configuration context.
	_, user := r.SessionManager.Logged(ctx)
	cctx, err := r.ConfigManager.CreateContext(int64(user.ID))
	if err != nil {
		msg := "Problem with creating transaction context"
		log.WithError(err).Error(msg)
		return nil, nil, http.StatusInternalServerError, msg
	}
	return respDaemons, cctx, 0, ""
}

// Common function that implements the POST calls to apply and commit a new
// or updated subnet. The ctx parameter is the REST API context. The
// transactionID is the identifier of the current configuration transaction
// used by the function to recover the transaction context. The restSubnet is
// the pointer to the subnet specified by the user. It is converted by this
// function to the database model. The applyFunc is the function of the Kea
// config module that applies the specified subnet. It is one of the ApplySubnetAdd
// or ApplySubnetUpdate, depending on whether the new subnet is created (via
// CreateSubnetSubmit) or updated (via UpdateSubnetSubmit). The apply functions
// receive the transaction context and a pointer to the subnet. They return the
// updated context and error. This function returns the HTTP error code if an
// error occurs or 0 when there is no error. In addition it returns an error
// string to be included in the HTTP response or an empty string if there is no
// error.
func (r *RestAPI) commonCreateOrUpdateSubnetSubmit(ctx context.Context, transactionID int64, restSubnet *models.Subnet, applyFunc func(context.Context, *dbmodel.Subnet) (context.Context, error)) (int, string) {
	// Make sure that the subnet information is present.
	if restSubnet == nil {
		msg := "Subnet information not specified"
		log.Errorf("Problem with submitting a subnet because the subnet information is missing")
		return http.StatusBadRequest, msg
	}
	// Retrieve the context from the config manager.
	_, user := r.SessionManager.Logged(ctx)
	cctx, _ := r.ConfigManager.RecoverContext(transactionID, int64(user.ID))
	if cctx == nil {
		msg := "Transaction expired for the subnet update"
		log.Errorf("Problem with recovering transaction context for transaction ID %d and user ID %d", transactionID, user.ID)
		return http.StatusNotFound, msg
	}

	// Convert subnet information from REST API to database format.
	subnet, err := r.convertSubnetFromRestAPI(restSubnet)
	if err != nil {
		msg := "Error parsing specified subnet"
		log.WithError(err).Error(msg)
		return http.StatusBadRequest, msg
	}
	err = subnet.PopulateDaemons(r.DB)
	if err != nil {
		msg := "Specified subnet is associated with daemons that no longer exist"
		log.WithError(err).Error(err)
		return http.StatusNotFound, msg
	}
	// Apply the subnet information (create Kea commands).
	cctx, err = applyFunc(cctx, subnet)
	if err != nil {
		msg := "Problem with applying subnet information"
		log.WithError(err).Error(msg)
		return http.StatusInternalServerError, msg
	}
	// Send the commands to Kea servers.
	cctx, err = r.ConfigManager.Commit(cctx)
	if err != nil {
		msg := fmt.Sprintf("Problem with committing subnet information: %s", err)
		log.WithError(err).Error(msg)
		return http.StatusConflict, msg
	}
	// Everything ok. Cleanup and send OK to the client.
	r.ConfigManager.Done(cctx)
	return 0, ""
}

// Common function that implements the DELETE calls to cancel adding new
// or updating a subnet. It removes the specified transaction from the
// config manager, if the transaction exists. It returns the HTTP error code
// if an error occurs or 0 when there is no error. In addition it returns an
// error string to be included in the HTTP response or an empty string if there
// is no error.
func (r *RestAPI) commonCreateOrUpdateSubnetDelete(ctx context.Context, transactionID int64) (int, string) {
	// Retrieve the context from the config manager.
	_, user := r.SessionManager.Logged(ctx)
	cctx, _ := r.ConfigManager.RecoverContext(transactionID, int64(user.ID))
	if cctx == nil {
		msg := "Transaction expired for the subnet update"
		log.Errorf("Problem with recovering transaction context for transaction ID %d and user ID %d", transactionID, user.ID)
		return http.StatusNotFound, msg
	}
	r.ConfigManager.Done(cctx)
	return 0, ""
}

// Implements the POST call to create new transaction for updating an
// existing subnet (subnets/{subnetId}/transaction).
func (r *RestAPI) UpdateSubnetBegin(ctx context.Context, params dhcp.UpdateSubnetBeginParams) middleware.Responder {
	// Execute the common part between create and update operations. It retrieves
	// the daemons and creates a transaction context.
	respDaemons, cctx, code, msg := r.commonCreateOrUpdateSubnetBegin(ctx)
	if code != 0 {
		// Error case.
		rsp := dhcp.NewUpdateSubnetBeginDefault(code).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Begin subnet update transaction. It retrieves current subnet information and
	// locks daemons for updates.
	var err error
	cctx, err = r.ConfigManager.GetKeaModule().BeginSubnetUpdate(cctx, params.SubnetID)
	if err != nil {
		var (
			subnetNotFound     *config.SubnetNotFoundError
			lock               *config.LockError
			hooksNotConfigured *config.NoSubnetCmdsHookError
		)
		switch {
		case errors.As(err, &subnetNotFound):
			// Failed to find subnet.
			msg := fmt.Sprintf("Unable to edit the subnet with ID %d because it cannot be found", params.SubnetID)
			log.Error(msg)
			rsp := dhcp.NewUpdateSubnetBeginDefault(http.StatusBadRequest).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		case errors.As(err, &lock):
			// Failed to lock daemons.
			msg := fmt.Sprintf("Unable to edit the subnet with ID %d because it may be currently edited by another user", params.SubnetID)
			log.WithError(err).Error(msg)
			rsp := dhcp.NewUpdateSubnetBeginDefault(http.StatusLocked).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		case errors.As(err, &hooksNotConfigured):
			// Lack of the libdhcp_subnet_cmds hook.
			msg := "Unable to update subnet configuration because some daemons lack libdhcp_subnet_cmds hook library"
			log.Error(msg)
			rsp := dhcp.NewUpdateSubnetBeginDefault(http.StatusBadRequest).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		default:
			// Other error.
			msg := fmt.Sprintf("Problem with initializing transaction for an update of the subnet with ID %d", params.SubnetID)
			log.WithError(err).Error(msg)
			rsp := dhcp.NewUpdateSubnetBeginDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
	}
	state, _ := config.GetTransactionState[kea.ConfigRecipe](cctx)
	subnet := state.Updates[0].Recipe.SubnetBeforeUpdate

	// Retrieve the generated context ID.
	cctxID, ok := config.GetValueAsInt64(cctx, config.ContextIDKey)
	if !ok {
		msg := "problem with retrieving context ID for a transaction to update a subnet"
		log.Error(msg)
		rsp := dhcp.NewUpdateSubnetBeginDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Remember the context, i.e. new transaction has been successfully created.
	_ = r.ConfigManager.RememberContext(cctx, time.Minute*10)

	// Return transaction ID and daemons to the user.
	contents := &models.UpdateSubnetBeginResponse{
		ID:      cctxID,
		Subnet:  r.convertSubnetToRestAPI(subnet),
		Daemons: respDaemons,
	}
	rsp := dhcp.NewUpdateSubnetBeginOK().WithPayload(contents)
	return rsp
}

// Implements the POST call and commits an updated subnet (subnets/{subnetId}/transaction/{id}/submit).
func (r *RestAPI) UpdateSubnetSubmit(ctx context.Context, params dhcp.UpdateSubnetSubmitParams) middleware.Responder {
	if code, msg := r.commonCreateOrUpdateSubnetSubmit(ctx, params.ID, params.Subnet, r.ConfigManager.GetKeaModule().ApplySubnetUpdate); code != 0 {
		// Error case.
		rsp := dhcp.NewUpdateSubnetSubmitDefault(code).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	rsp := dhcp.NewUpdateSubnetSubmitOK()
	return rsp
}

// Implements the DELETE call to cancel updating a subnet (subnets/{subnetId}/transaction/{id}).
// It removes the specified transaction from the config manager, if the transaction exists.
func (r *RestAPI) UpdateSubnetDelete(ctx context.Context, params dhcp.UpdateSubnetDeleteParams) middleware.Responder {
	if code, msg := r.commonCreateOrUpdateSubnetDelete(ctx, params.ID); code != 0 {
		// Error case.
		rsp := dhcp.NewUpdateSubnetDeleteDefault(code).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	rsp := dhcp.NewUpdateSubnetDeleteOK()
	return rsp
}
