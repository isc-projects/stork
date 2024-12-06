package restservice

import (
	"context"
	"fmt"
	"net/http"
	"sort"
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

// Converts Kea specific global parameters derived to a subnet or shared network
// to a format used in the REST API.
func convertGlobalSubnetParametersToRestAPI(cfg *dbmodel.KeaConfig) *models.KeaConfigSubnetDerivedParameters {
	return &models.KeaConfigSubnetDerivedParameters{
		KeaConfigCacheParameters: models.KeaConfigCacheParameters{
			CacheThreshold: cfg.GetCacheParameters().CacheThreshold,
			CacheMaxAge:    cfg.GetCacheParameters().CacheMaxAge,
		},
		KeaConfigDdnsParameters: models.KeaConfigDdnsParameters{
			DdnsGeneratedPrefix:        storkutil.NullifyEmptyString(cfg.GetDDNSParameters().DDNSGeneratedPrefix),
			DdnsOverrideClientUpdate:   cfg.GetDDNSParameters().DDNSOverrideClientUpdate,
			DdnsOverrideNoUpdate:       cfg.GetDDNSParameters().DDNSOverrideNoUpdate,
			DdnsQualifyingSuffix:       storkutil.NullifyEmptyString(cfg.GetDDNSParameters().DDNSQualifyingSuffix),
			DdnsReplaceClientName:      storkutil.NullifyEmptyString(cfg.GetDDNSParameters().DDNSReplaceClientName),
			DdnsSendUpdates:            cfg.GetDDNSParameters().DDNSSendUpdates,
			DdnsUpdateOnRenew:          cfg.GetDDNSParameters().DDNSUpdateOnRenew,
			DdnsUseConflictResolution:  cfg.GetDDNSParameters().DDNSUseConflictResolution,
			DdnsConflictResolutionMode: cfg.GetDDNSParameters().DDNSConflictResolutionMode,
			DdnsTTLPercent:             cfg.GetDDNSParameters().DDNSTTLPercent,
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
}

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
			UserContext:      lsn.UserContext,
		}
		for _, poolDetails := range lsn.AddressPools {
			pool := &models.Pool{
				Pool: storkutil.Ptr(poolDetails.LowerBound + "-" + poolDetails.UpperBound),
			}
			if poolDetails.KeaParameters != nil {
				pool.KeaConfigPoolParameters = &models.KeaConfigPoolParameters{
					KeaConfigAssortedPoolParameters: models.KeaConfigAssortedPoolParameters{
						PoolID: poolDetails.KeaParameters.PoolID,
					},
					KeaConfigClientClassParameters: models.KeaConfigClientClassParameters{
						ClientClass:          storkutil.NullifyEmptyString(poolDetails.KeaParameters.ClientClass),
						RequireClientClasses: poolDetails.KeaParameters.RequireClientClasses,
					},
				}
				// DHCP options.
				pool.KeaConfigPoolParameters.OptionsHash = poolDetails.DHCPOptionSet.Hash
				pool.KeaConfigPoolParameters.Options = r.unflattenDHCPOptions(poolDetails.DHCPOptionSet.Options, "", 0)
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
					KeaConfigAssortedPoolParameters: models.KeaConfigAssortedPoolParameters{
						PoolID: prefixPoolDetails.KeaParameters.PoolID,
					},
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
					DdnsGeneratedPrefix:        storkutil.NullifyEmptyString(keaParameters.DDNSGeneratedPrefix),
					DdnsOverrideClientUpdate:   keaParameters.DDNSOverrideClientUpdate,
					DdnsOverrideNoUpdate:       keaParameters.DDNSOverrideNoUpdate,
					DdnsQualifyingSuffix:       storkutil.NullifyEmptyString(keaParameters.DDNSQualifyingSuffix),
					DdnsReplaceClientName:      storkutil.NullifyEmptyString(keaParameters.DDNSReplaceClientName),
					DdnsSendUpdates:            keaParameters.DDNSSendUpdates,
					DdnsUpdateOnRenew:          keaParameters.DDNSUpdateOnRenew,
					DdnsUseConflictResolution:  keaParameters.DDNSUseConflictResolution,
					DdnsConflictResolutionMode: keaParameters.DDNSConflictResolutionMode,
					DdnsTTLPercent:             keaParameters.DDNSTTLPercent,
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
			localSubnet.KeaConfigSubnetParameters.SubnetLevelParameters.OptionsHash = lsn.DHCPOptionSet.Hash
			localSubnet.KeaConfigSubnetParameters.SubnetLevelParameters.Options = r.unflattenDHCPOptions(lsn.DHCPOptionSet.Options, "", 0)
		}
		// Shared network level Kea DHCP parameters.
		if sn.SharedNetwork != nil {
			keaParameters := sn.SharedNetwork.GetKeaParameters(lsn.DaemonID)
			if keaParameters != nil {
				if localSubnet.KeaConfigSubnetParameters == nil {
					localSubnet.KeaConfigSubnetParameters = &models.KeaConfigSubnetParameters{}
				}
				localSubnet.KeaConfigSubnetParameters.SharedNetworkLevelParameters = convertSharedNetworkParametersToRestAPI(keaParameters)
				if localSharedNetwork := sn.SharedNetwork.GetLocalSharedNetwork(lsn.DaemonID); localSharedNetwork != nil {
					localSubnet.KeaConfigSubnetParameters.SharedNetworkLevelParameters.OptionsHash = localSharedNetwork.DHCPOptionSet.Hash
					localSubnet.KeaConfigSubnetParameters.SharedNetworkLevelParameters.Options = r.unflattenDHCPOptions(localSharedNetwork.DHCPOptionSet.Options, "", 0)
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
			localSubnet.KeaConfigSubnetParameters.GlobalParameters = convertGlobalSubnetParametersToRestAPI(cfg)
			var convertedOptions []dbmodel.DHCPOption
			for _, option := range cfg.GetDHCPOptions() {
				convertedOption, err := dbmodel.NewDHCPOptionFromKea(option, storkutil.IPType(sn.GetFamily()), r.DHCPOptionDefinitionLookup)
				if err != nil {
					continue
				}
				convertedOptions = append(convertedOptions, *convertedOption)
			}
			localSubnet.KeaConfigSubnetParameters.GlobalParameters.OptionsHash = keaconfig.NewHasher().Hash(convertedOptions)
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
	hasher := keaconfig.NewHasher()
	// Convert local subnet containing associations of the subnet with daemons.
	for _, ls := range restSubnet.LocalSubnets {
		localSubnet := &dbmodel.LocalSubnet{
			LocalSubnetID: ls.ID,
			DaemonID:      ls.DaemonID,
		}

		if userContext, ok := ls.UserContext.(map[string]any); ok {
			localSubnet.UserContext = userContext
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
					PoolID: poolDetails.KeaConfigPoolParameters.PoolID,
				}
				// DHCP options.
				options, err := r.flattenDHCPOptions("", poolDetails.KeaConfigPoolParameters.Options, 0)
				if err != nil {
					return nil, err
				}
				pool.SetDHCPOptions(options, hasher)
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
					PoolID: prefixPoolDetails.KeaConfigPoolParameters.PoolID,
				}
				// DHCP options.
				pool.DHCPOptionSet, err = r.flattenDHCPOptions("", prefixPoolDetails.KeaConfigPoolParameters.Options, 0)
				if err != nil {
					return nil, err
				}
				if len(pool.DHCPOptionSet) > 0 {
					pool.DHCPOptionSetHash = hasher.Hash(pool.DHCPOptionSet)
				}
			}
			localSubnet.PrefixPools = append(localSubnet.PrefixPools, *pool)
		}

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
					DDNSGeneratedPrefix:        storkutil.NullifyEmptyString(keaParameters.DdnsGeneratedPrefix),
					DDNSOverrideClientUpdate:   keaParameters.DdnsOverrideClientUpdate,
					DDNSOverrideNoUpdate:       keaParameters.DdnsOverrideNoUpdate,
					DDNSQualifyingSuffix:       storkutil.NullifyEmptyString(keaParameters.DdnsQualifyingSuffix),
					DDNSReplaceClientName:      storkutil.NullifyEmptyString(keaParameters.DdnsReplaceClientName),
					DDNSSendUpdates:            keaParameters.DdnsSendUpdates,
					DDNSUpdateOnRenew:          keaParameters.DdnsUpdateOnRenew,
					DDNSUseConflictResolution:  keaParameters.DdnsUseConflictResolution,
					DDNSConflictResolutionMode: keaParameters.DdnsConflictResolutionMode,
					DDNSTTLPercent:             keaParameters.DdnsTTLPercent,
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
			options, err := r.flattenDHCPOptions("", ls.KeaConfigSubnetParameters.SubnetLevelParameters.Options, 0)
			if err != nil {
				return nil, err
			}
			localSubnet.DHCPOptionSet.SetDHCPOptions(options, keaconfig.NewHasher())
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

// Common function executed when creating a new transaction for when the
// subnet or a shared network is created or updated. It fetches available
// DHCP daemons. It also creates transaction context. If an error occurs,
// an http error code and message are returned.
func (r *RestAPI) commonCreateOrUpdateNetworkBegin(ctx context.Context) ([]*models.KeaDaemon, []*models.SharedNetwork, []*models.SharedNetwork, []string, context.Context, int, string) {
	// A list of Kea DHCP daemons will be needed in the user form,
	// so the user can select which servers send the subnet to.
	daemons, err := dbmodel.GetKeaDHCPDaemons(r.DB)
	if err != nil {
		msg := "Problem with fetching Kea daemons from the database"
		log.Error(err)
		return nil, nil, nil, nil, nil, http.StatusInternalServerError, msg
	}
	sharedNetworks, err := dbmodel.GetAllSharedNetworks(r.DB, 0)
	if err != nil {
		msg := "Problem with fetching shared networks from the database"
		log.WithError(err).Error(msg)
		return nil, nil, nil, nil, nil, http.StatusInternalServerError, msg
	}
	// Convert daemons list to REST API format and extract their configured
	// client classes.
	respDaemons := []*models.KeaDaemon{}
	respClientClasses := []string{}
	clientClassesMap := make(map[string]bool)
	for i := range daemons {
		if daemons[i].KeaDaemon != nil && daemons[i].KeaDaemon.Config != nil {
			// Filter the daemons with subnet_cmds hook library.
			if _, _, exists := daemons[i].KeaDaemon.Config.GetHookLibrary("libdhcp_subnet_cmds"); exists {
				respDaemons = append(respDaemons, keaDaemonToRestAPI(&daemons[i]))
			}
			clientClasses := daemons[i].KeaDaemon.Config.GetClientClasses()
			for _, c := range clientClasses {
				clientClassesMap[c.Name] = true
			}
		}
	}
	// Turn the class map to a slice and sort it by a class name.
	for c := range clientClassesMap {
		respClientClasses = append(respClientClasses, c)
	}
	sort.Strings(respClientClasses)

	// Append shared networks list.
	respIPv4SharedNetworks := []*models.SharedNetwork{}
	respIPv6SharedNetworks := []*models.SharedNetwork{}
	for i := range sharedNetworks {
		respSharedNetwork := r.convertSharedNetworkToRestAPI(&sharedNetworks[i])
		switch respSharedNetwork.Universe {
		case 4:
			respIPv4SharedNetworks = append(respIPv4SharedNetworks, respSharedNetwork)
		default:
			respIPv6SharedNetworks = append(respIPv6SharedNetworks, respSharedNetwork)
		}
	}

	// If there are no daemons with subnet_cmds hooks library loaded there is no way
	// to add new host reservation. In that case, we don't begin a transaction.
	if len(respDaemons) == 0 {
		msg := "Unable to begin transaction because there are no Kea servers with subnet_cmds hooks library available"
		log.Error(msg)
		return nil, nil, nil, nil, nil, http.StatusBadRequest, msg
	}
	// Create configuration context.
	_, user := r.SessionManager.Logged(ctx)
	cctx, err := r.ConfigManager.CreateContext(int64(user.ID))
	if err != nil {
		msg := "Problem with creating transaction context"
		log.WithError(err).Error(msg)
		return nil, nil, nil, nil, nil, http.StatusInternalServerError, msg
	}
	return respDaemons, respIPv4SharedNetworks, respIPv6SharedNetworks, respClientClasses, cctx, 0, ""
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
// error occurs or 0 when there is no error. It also returns an ID of the
// created or modified subnet. Finally, it returns an error string to be included
// in the HTTP response or an empty string if there is no error.
func (r *RestAPI) commonCreateOrUpdateSubnetSubmit(ctx context.Context, transactionID int64, restSubnet *models.Subnet, applyFunc func(context.Context, *dbmodel.Subnet) (context.Context, error)) (int, int64, string) {
	// Make sure that the subnet information is present.
	if restSubnet == nil {
		msg := "Subnet information not specified"
		log.Errorf("Problem with submitting a subnet because the subnet information is missing")
		return http.StatusBadRequest, 0, msg
	}
	// Retrieve the context from the config manager.
	_, user := r.SessionManager.Logged(ctx)
	cctx, _ := r.ConfigManager.RecoverContext(transactionID, int64(user.ID))
	if cctx == nil {
		msg := "Transaction expired for the subnet update"
		log.Errorf("Problem with recovering transaction context for transaction ID %d and user ID %d", transactionID, user.ID)
		return http.StatusNotFound, 0, msg
	}

	// Convert subnet information from REST API to database format.
	subnet, err := r.convertSubnetFromRestAPI(restSubnet)
	if err != nil {
		msg := "Error parsing specified subnet"
		log.WithError(err).Error(msg)
		return http.StatusBadRequest, 0, msg
	}
	err = subnet.PopulateDaemons(r.DB)
	if err != nil {
		msg := "Specified subnet is associated with daemons that no longer exist"
		log.WithError(err).Error(msg)
		return http.StatusNotFound, 0, msg
	}
	if restSubnet.SharedNetwork != "" {
		subnet.SharedNetwork = &dbmodel.SharedNetwork{
			Name: restSubnet.SharedNetwork,
		}
	}
	// Apply the subnet information (create Kea commands).
	cctx, err = applyFunc(cctx, subnet)
	if err != nil {
		msg := fmt.Sprintf("Problem with applying subnet information: %s", err)
		log.WithError(err).Error(msg)
		return http.StatusInternalServerError, 0, msg
	}
	// Send the commands to Kea servers.
	cctx, err = r.ConfigManager.Commit(cctx)
	if err != nil {
		msg := fmt.Sprintf("Problem with committing subnet information: %s", err)
		log.WithError(err).Error(msg)
		return http.StatusConflict, 0, msg
	}
	subnetID := restSubnet.ID
	if subnetID == 0 {
		recipe, err := config.GetRecipeForUpdate[kea.ConfigRecipe](cctx, 0)
		if err != nil {
			msg := "Problem recovering subnet ID from the context"
			log.WithError(err).Error(msg)
			return http.StatusInternalServerError, 0, msg
		}
		if recipe.SubnetID != nil {
			subnetID = *recipe.SubnetID
		}
	}
	// Everything ok. Cleanup and send OK to the client.
	r.ConfigManager.Done(cctx)
	return 0, subnetID, ""
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

// Implements the POST call to create new transaction for adding a new
// subnet (subnets/new/transaction).
func (r *RestAPI) CreateSubnetBegin(ctx context.Context, params dhcp.CreateSubnetBeginParams) middleware.Responder {
	// Execute the common part between create and update operations. It retrieves
	// the daemons and creates a transaction context.
	respDaemons, respIPv4SharedNetworks, respIPv6SharedNetworks, respClientClasses, cctx, code, msg := r.commonCreateOrUpdateNetworkBegin(ctx)
	if code != 0 {
		// Error case.
		rsp := dhcp.NewCreateSubnetBeginDefault(code).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	respSubnets, err := dbmodel.GetSubnetPrefixes(r.DB)
	if err != nil {
		msg := "Problem with fetching subnets from the database"
		log.WithError(err).Error(msg)
		rsp := dhcp.NewCreateSubnetBeginDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Begin subnet add transaction.
	if cctx, err = r.ConfigManager.GetKeaModule().BeginSubnetAdd(cctx); err != nil {
		msg := "Problem with initializing transaction for creating subnet"
		log.WithError(err).Error(msg)
		rsp := dhcp.NewCreateSubnetBeginDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// Retrieve the generated context ID.
	cctxID, ok := config.GetValueAsInt64(cctx, config.ContextIDKey)
	if !ok {
		msg := "problem with retrieving context ID for a transaction to create a subnet"
		log.Error(msg)
		rsp := dhcp.NewCreateSubnetBeginDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Remember the context, i.e. new transaction has been successfully created.
	_ = r.ConfigManager.RememberContext(cctx, time.Minute*10)

	// Return transaction ID and daemons to the user.
	contents := &models.CreateSubnetBeginResponse{
		ID:              cctxID,
		Daemons:         respDaemons,
		SharedNetworks4: respIPv4SharedNetworks,
		SharedNetworks6: respIPv6SharedNetworks,
		Subnets:         respSubnets,
		ClientClasses:   respClientClasses,
	}
	rsp := dhcp.NewCreateSubnetBeginOK().WithPayload(contents)
	return rsp
}

// Implements the POST call and commits a new subnet (subnets/new/transaction/{id}/submit).
func (r *RestAPI) CreateSubnetSubmit(ctx context.Context, params dhcp.CreateSubnetSubmitParams) middleware.Responder {
	code, subnetID, msg := r.commonCreateOrUpdateSubnetSubmit(ctx, params.ID, params.Subnet, r.ConfigManager.GetKeaModule().ApplySubnetAdd)
	if code != 0 {
		// Error case.
		rsp := dhcp.NewCreateSubnetSubmitDefault(code).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	contents := &models.CreateSubnetSubmitResponse{
		SubnetID: subnetID,
	}
	rsp := dhcp.NewCreateSubnetSubmitOK().WithPayload(contents)
	return rsp
}

// Implements the DELETE call to cancel creating a subnet (subnets/new/transaction/{id}).
// It removes the specified transaction from the config manager, if the transaction exists.
func (r *RestAPI) CreateSubnetDelete(ctx context.Context, params dhcp.CreateSubnetDeleteParams) middleware.Responder {
	if code, msg := r.commonCreateOrUpdateSubnetDelete(ctx, params.ID); code != 0 {
		// Error case.
		rsp := dhcp.NewCreateSubnetDeleteDefault(code).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	rsp := dhcp.NewCreateSubnetDeleteOK()
	return rsp
}

// Implements the POST call to create new transaction for updating an
// existing subnet (subnets/{subnetId}/transaction).
func (r *RestAPI) UpdateSubnetBegin(ctx context.Context, params dhcp.UpdateSubnetBeginParams) middleware.Responder {
	// Execute the common part between create and update operations. It retrieves
	// the daemons and creates a transaction context.
	respDaemons, respIPv4SharedNetworks, respIPv6SharedNetworks, respClientClasses, cctx, code, msg := r.commonCreateOrUpdateNetworkBegin(ctx)
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
		ID:              cctxID,
		Subnet:          r.convertSubnetToRestAPI(subnet),
		Daemons:         respDaemons,
		SharedNetworks4: respIPv4SharedNetworks,
		SharedNetworks6: respIPv6SharedNetworks,
		ClientClasses:   respClientClasses,
	}
	rsp := dhcp.NewUpdateSubnetBeginOK().WithPayload(contents)
	return rsp
}

// Implements the POST call and commits an updated subnet (subnets/{subnetId}/transaction/{id}/submit).
func (r *RestAPI) UpdateSubnetSubmit(ctx context.Context, params dhcp.UpdateSubnetSubmitParams) middleware.Responder {
	if code, _, msg := r.commonCreateOrUpdateSubnetSubmit(ctx, params.ID, params.Subnet, r.ConfigManager.GetKeaModule().ApplySubnetUpdate); code != 0 {
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

// Implements the DELETE call for a subnet (subnets/{id}). It sends suitable commands
// to the Kea servers owning the subnet. Deleting subnet is not transactional. It could be
// implemented as a transaction with first REST API call ensuring that the subnet still
// exists in Stork database and locking configuration changes for the daemons owning the
// subnet. However, it seems to be too much overhead with little gain. If the subnet
// doesn't exist this call will return an error anyway.
func (r *RestAPI) DeleteSubnet(ctx context.Context, params dhcp.DeleteSubnetParams) middleware.Responder {
	dbSubnet, err := dbmodel.GetSubnet(r.DB, params.ID)
	if err != nil {
		// Error while communicating with the database.
		msg := fmt.Sprintf("Problem fetching subnet with ID %d from db", params.ID)
		log.WithError(err).Error(msg)
		rsp := dhcp.NewDeleteSubnetDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbSubnet == nil {
		// Host not found.
		msg := fmt.Sprintf("Cannot find a subnet with ID %d", params.ID)
		rsp := dhcp.NewDeleteSubnetDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Create configuration context.
	_, user := r.SessionManager.Logged(ctx)
	cctx, err := r.ConfigManager.CreateContext(int64(user.ID))
	if err != nil {
		msg := "Problem with creating transaction context for deleting the subnet"
		log.WithError(err).Error(err)
		rsp := dhcp.NewDeleteSubnetDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Create Kea commands to delete the subnet.
	cctx, err = r.ConfigManager.GetKeaModule().ApplySubnetDelete(cctx, dbSubnet)
	if err != nil {
		msg := "Problem with preparing commands for deleting the subnet"
		log.WithError(err).Error(msg)
		rsp := dhcp.NewDeleteSubnetDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Send the commands to Kea servers.
	_, err = r.ConfigManager.Commit(cctx)
	if err != nil {
		msg := fmt.Sprintf("Problem with deleting a subnet: %s", err)
		log.WithError(err).Error(msg)
		rsp := dhcp.NewDeleteSubnetDefault(http.StatusConflict).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Send OK to the client.
	rsp := dhcp.NewDeleteSubnetOK()
	return rsp
}
