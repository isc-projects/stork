package restservice

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	log "github.com/sirupsen/logrus"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"

	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
)

// Creates a REST API representation of a subnet from a database model.
func (r *RestAPI) subnetToRestAPI(sn *dbmodel.Subnet) *models.Subnet {
	subnet := &models.Subnet{
		ID:               sn.ID,
		Subnet:           sn.Prefix,
		ClientClass:      sn.ClientClass,
		AddrUtilization:  float64(sn.AddrUtilization) / 10,
		PdUtilization:    float64(sn.PdUtilization) / 10,
		Stats:            sn.Stats,
		StatsCollectedAt: strfmt.DateTime(sn.StatsCollectedAt),
	}

	if sn.SharedNetwork != nil {
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
			StatsCollectedAt: strfmt.DateTime(lsn.StatsCollectedAt),
		}
		for _, poolDetails := range lsn.AddressPools {
			pool := poolDetails.LowerBound + "-" + poolDetails.UpperBound
			localSubnet.Pools = append(localSubnet.Pools, pool)
		}

		for _, prefixPoolDetails := range lsn.PrefixPools {
			prefix := prefixPoolDetails.Prefix
			delegatedLength := int64(prefixPoolDetails.DelegatedLen)
			localSubnet.PrefixDelegationPools = append(
				localSubnet.PrefixDelegationPools,
				&models.DelegatedPrefix{
					Prefix:          &prefix,
					DelegatedLength: &delegatedLength,
					ExcludedPrefix:  prefixPoolDetails.ExcludedPrefix,
				},
			)
		}

		// Subnet level Kea DHCP parameters.
		if lsn.KeaParameters != nil {
			keaParameters := lsn.KeaParameters
			localSubnet.KeaConfigSubnetParameters = &models.KeaConfigSubnetParameters{}
			localSubnet.KeaConfigSubnetParameters.SubnetLevelParameters = &models.KeaConfigSubnetDerivedParameters{
				KeaConfigCacheParameters: models.KeaConfigCacheParameters{
					CacheThreshold: keaParameters.CacheThreshold,
					CacheMaxAge:    keaParameters.CacheMaxAge,
				},
				KeaConfigClientClassParameters: models.KeaConfigClientClassParameters{
					ClientClass:          keaParameters.ClientClass,
					RequireClientClasses: keaParameters.RequireClientClasses,
				},
				KeaConfigDdnsParameters: models.KeaConfigDdnsParameters{
					DdnsGeneratedPrefix:       keaParameters.DDNSGeneratedPrefix,
					DdnsOverrideClientUpdate:  keaParameters.DDNSOverrideClientUpdate,
					DdnsOverrideNoUpdate:      keaParameters.DDNSOverrideNoUpdate,
					DdnsQualifyingSuffix:      keaParameters.DDNSQualifyingSuffix,
					DdnsReplaceClientName:     keaParameters.DDNSReplaceClientName,
					DdnsSendUpdates:           keaParameters.DDNSSendUpdates,
					DdnsUpdateOnRenew:         keaParameters.DDNSUpdateOnRenew,
					DdnsUseConflictResolution: keaParameters.DDNSUseConflictResolution,
				},
				KeaConfigFourOverSixParameters: models.KeaConfigFourOverSixParameters{
					FourOverSixInterface:   keaParameters.FourOverSixInterface,
					FourOverSixInterfaceID: keaParameters.FourOverSixInterfaceID,
					FourOverSixSubnet:      keaParameters.FourOverSixSubnet,
				},
				KeaConfigHostnameCharParameters: models.KeaConfigHostnameCharParameters{
					HostnameCharReplacement: keaParameters.HostnameCharReplacement,
					HostnameCharSet:         keaParameters.HostnameCharSet,
				},
				KeaConfigPreferredLifetimeParameters: models.KeaConfigPreferredLifetimeParameters{
					MaxPreferredLifetime: keaParameters.MaxPreferredLifetime,
					MinPreferredLifetime: keaParameters.MinPreferredLifetime,
					PreferredLifetime:    keaParameters.PreferredLifetime,
				},
				KeaConfigReservationParameters: models.KeaConfigReservationParameters{
					ReservationMode:       keaParameters.ReservationMode,
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
					Allocator:         keaParameters.Allocator,
					Authoritative:     keaParameters.Authoritative,
					BootFileName:      keaParameters.BootFileName,
					Interface:         keaParameters.Interface,
					InterfaceID:       keaParameters.InterfaceID,
					MatchClientID:     keaParameters.MatchClientID,
					NextServer:        keaParameters.NextServer,
					PdAllocator:       keaParameters.PDAllocator,
					RapidCommit:       keaParameters.RapidCommit,
					ServerHostname:    keaParameters.ServerHostname,
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
				localSubnet.KeaConfigSubnetParameters.SharedNetworkLevelParameters = &models.KeaConfigSubnetDerivedParameters{
					KeaConfigCacheParameters: models.KeaConfigCacheParameters{
						CacheThreshold: keaParameters.CacheThreshold,
						CacheMaxAge:    keaParameters.CacheMaxAge,
					},
					KeaConfigClientClassParameters: models.KeaConfigClientClassParameters{
						ClientClass:          keaParameters.ClientClass,
						RequireClientClasses: keaParameters.RequireClientClasses,
					},
					KeaConfigDdnsParameters: models.KeaConfigDdnsParameters{
						DdnsGeneratedPrefix:       keaParameters.DDNSGeneratedPrefix,
						DdnsOverrideClientUpdate:  keaParameters.DDNSOverrideClientUpdate,
						DdnsOverrideNoUpdate:      keaParameters.DDNSOverrideNoUpdate,
						DdnsQualifyingSuffix:      keaParameters.DDNSQualifyingSuffix,
						DdnsReplaceClientName:     keaParameters.DDNSReplaceClientName,
						DdnsSendUpdates:           keaParameters.DDNSSendUpdates,
						DdnsUpdateOnRenew:         keaParameters.DDNSUpdateOnRenew,
						DdnsUseConflictResolution: keaParameters.DDNSUseConflictResolution,
					},
					KeaConfigHostnameCharParameters: models.KeaConfigHostnameCharParameters{
						HostnameCharReplacement: keaParameters.HostnameCharReplacement,
						HostnameCharSet:         keaParameters.HostnameCharSet,
					},
					KeaConfigPreferredLifetimeParameters: models.KeaConfigPreferredLifetimeParameters{
						MaxPreferredLifetime: keaParameters.MaxPreferredLifetime,
						MinPreferredLifetime: keaParameters.MinPreferredLifetime,
						PreferredLifetime:    keaParameters.PreferredLifetime,
					},
					KeaConfigReservationParameters: models.KeaConfigReservationParameters{
						ReservationMode:       keaParameters.ReservationMode,
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
						Allocator:         keaParameters.Allocator,
						Authoritative:     keaParameters.Authoritative,
						BootFileName:      keaParameters.BootFileName,
						Interface:         keaParameters.Interface,
						InterfaceID:       keaParameters.InterfaceID,
						MatchClientID:     keaParameters.MatchClientID,
						NextServer:        keaParameters.NextServer,
						PdAllocator:       keaParameters.PDAllocator,
						RapidCommit:       keaParameters.RapidCommit,
						ServerHostname:    keaParameters.ServerHostname,
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
			localSubnet.KeaConfigSubnetParameters.GlobalParameters = &models.KeaConfigSubnetDerivedParameters{
				KeaConfigCacheParameters: models.KeaConfigCacheParameters{
					CacheThreshold: cfg.GetCacheParameters().CacheThreshold,
					CacheMaxAge:    cfg.GetCacheParameters().CacheMaxAge,
				},
				KeaConfigDdnsParameters: models.KeaConfigDdnsParameters{
					DdnsGeneratedPrefix:       cfg.GetDDNSParameters().DDNSGeneratedPrefix,
					DdnsOverrideClientUpdate:  cfg.GetDDNSParameters().DDNSOverrideClientUpdate,
					DdnsOverrideNoUpdate:      cfg.GetDDNSParameters().DDNSOverrideNoUpdate,
					DdnsQualifyingSuffix:      cfg.GetDDNSParameters().DDNSQualifyingSuffix,
					DdnsReplaceClientName:     cfg.GetDDNSParameters().DDNSReplaceClientName,
					DdnsSendUpdates:           cfg.GetDDNSParameters().DDNSSendUpdates,
					DdnsUpdateOnRenew:         cfg.GetDDNSParameters().DDNSUpdateOnRenew,
					DdnsUseConflictResolution: cfg.GetDDNSParameters().DDNSUseConflictResolution,
				},
				KeaConfigHostnameCharParameters: models.KeaConfigHostnameCharParameters{
					HostnameCharReplacement: cfg.GetHostnameCharParameters().HostnameCharReplacement,
					HostnameCharSet:         cfg.GetHostnameCharParameters().HostnameCharSet,
				},
				KeaConfigPreferredLifetimeParameters: models.KeaConfigPreferredLifetimeParameters{
					MaxPreferredLifetime: cfg.GetPreferredLifetimeParameters().MaxPreferredLifetime,
					MinPreferredLifetime: cfg.GetPreferredLifetimeParameters().MinPreferredLifetime,
					PreferredLifetime:    cfg.GetPreferredLifetimeParameters().PreferredLifetime,
				},
				KeaConfigReservationParameters: models.KeaConfigReservationParameters{
					ReservationMode:       cfg.GetGlobalReservationParameters().ReservationMode,
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
					Allocator:         cfg.GetAllocator(),
					Authoritative:     cfg.GetAuthoritative(),
					BootFileName:      cfg.GetBootFileName(),
					MatchClientID:     cfg.GetMatchClientID(),
					NextServer:        cfg.GetNextServer(),
					PdAllocator:       cfg.GetPDAllocator(),
					RapidCommit:       cfg.GetRapidCommit(),
					ServerHostname:    cfg.GetServerHostname(),
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
			localSubnet.KeaConfigSubnetParameters.GlobalParameters.OptionsHash = storkutil.Fnv128AnyValue(convertedOptions)
			localSubnet.KeaConfigSubnetParameters.GlobalParameters.Options = r.unflattenDHCPOptions(convertedOptions, "", 0)
		}
		subnet.LocalSubnets = append(subnet.LocalSubnets, localSubnet)
	}
	return subnet
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
		subnet := r.subnetToRestAPI(&sn)
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

	subnet := r.subnetToRestAPI(dbSubnet)
	rsp := dhcp.NewGetSubnetOK().WithPayload(subnet)
	return rsp
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
	for _, net := range dbSharedNetworks {
		if len(net.Subnets) == 0 || len(net.Subnets[0].LocalSubnets) == 0 {
			continue
		}
		subnets := []*models.Subnet{}
		// Exclude the subnets that are not attached to any app. This shouldn't
		// be the case but let's be safe.
		for _, snTmp := range net.Subnets {
			sn := snTmp
			subnet := r.subnetToRestAPI(&sn)
			subnets = append(subnets, subnet)
		}
		// Create shared network.
		sharedNetwork := &models.SharedNetwork{
			ID:               net.ID,
			Name:             net.Name,
			Subnets:          subnets,
			AddrUtilization:  float64(net.AddrUtilization) / 10,
			PdUtilization:    float64(net.PdUtilization) / 10,
			Stats:            net.Stats,
			StatsCollectedAt: strfmt.DateTime(net.StatsCollectedAt),
		}
		sharedNetworks.Items = append(sharedNetworks.Items, sharedNetwork)
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
