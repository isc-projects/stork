package kea

import (
	"testing"

	require "github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbmodeltest "isc.org/stork/server/database/model/test"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test/dbmodel"
)

// Multi step test verifying that services can be gradually created from the
// Kea apps being added to the database.
func TestDetectHAServices(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	kea, err := dbmodeltest.NewKea(db)
	require.NoError(t, err)

	dhcp4, err := kea.NewKeaDHCPv4Server()
	require.NoError(t, err)

	err = dhcp4.Configure(`{
		"Dhcp4": {
            "hooks-libraries": [
                {
                    "library": "libdhcp_ha.so",
                    "parameters": {
                        "high-availability": [{
                            "this-server-name": "server2",
                            "mode": "load-balancing",
                            "peers": [
								{
									"name": "server1",
									"url":  "http://192.0.2.33:8000",
									"role": "primary"
								},
								{
									"name": "server2",
									"url":  "http://192.0.2.66:8000",
									"role": "secondary"
								},
								{
									"name": "server4",
									"url":  "http://192.0.2.133:8000",
									"role": "backup"
								}
							]
                        }]
                    }
                }
            ]
		}
	}`)
	require.NoError(t, err)

	dhcp6, err := kea.NewKeaDHCPv6Server()
	require.NoError(t, err)

	err = dhcp6.Configure(`{
		"Dhcp6": {
            "hooks-libraries": [
                {
                    "library": "libdhcp_ha.so",
                    "parameters": {
                        "high-availability": [{
                            "this-server-name": "server3",
                            "mode": "hot-standby",
                            "peers": [
								{
									"name": "server1",
									"url":  "http://192.0.2.33:8000",
									"role": "primary"
								},
								{
									"name": "server3",
									"url":  "http://192.0.2.66:8000",
									"role": "standby"
								}
							]
                        }]
                    }
                }
            ]
		}
	}`)
	require.NoError(t, err)

	app, err := dhcp4.GetKea()
	require.NoError(t, err)

	// Run service detection for all daemons in this app.
	var services []dbmodel.Service
	for i := range app.Daemons {
		detected, err := DetectHAServices(db, app.Daemons[i])
		require.NoError(t, err)
		services = append(services, detected...)
	}

	// There should be two services returned, one for DHCPv4 and one for DHCPv6.
	require.Len(t, services, 2)

	// Check the DHCPv4 service first.
	require.True(t, services[0].IsNew())
	require.NotNil(t, services[0].HAService)
	require.Equal(t, "dhcp4", services[0].HAService.HAType)
	require.Equal(t, "load-balancing", services[0].HAService.HAMode)
	require.Equal(t, "server2", services[0].HAService.Relationship)
	require.Zero(t, services[0].HAService.PrimaryID)
	require.Equal(t, app.Daemons[0].ID, services[0].HAService.SecondaryID)
	require.Empty(t, services[0].HAService.BackupID)
	require.True(t, services[0].HAService.PrimaryStatusCollectedAt.IsZero())
	require.True(t, services[0].HAService.SecondaryStatusCollectedAt.IsZero())
	require.Empty(t, services[0].HAService.PrimaryLastState)
	require.Empty(t, services[0].HAService.SecondaryLastState)

	// Check the DHCPv6 service.
	require.True(t, services[1].IsNew())
	require.NotNil(t, services[1].HAService)
	require.Equal(t, "dhcp6", services[1].HAService.HAType)
	require.Equal(t, "hot-standby", services[1].HAService.HAMode)
	require.Equal(t, "server2", services[0].HAService.Relationship)
	require.Zero(t, services[1].HAService.PrimaryID)
	require.Equal(t, app.Daemons[1].ID, services[1].HAService.SecondaryID)
	require.Empty(t, services[1].HAService.BackupID)
	require.True(t, services[1].HAService.PrimaryStatusCollectedAt.IsZero())
	require.True(t, services[1].HAService.SecondaryStatusCollectedAt.IsZero())
	require.Empty(t, services[1].HAService.PrimaryLastState)
	require.Empty(t, services[1].HAService.SecondaryLastState)

	// These are new services, so the daemons should have been added to them.
	require.Len(t, services[0].Daemons, 1)
	require.Len(t, services[1].Daemons, 1)

	// Add the services to the database.
	err = dbmodel.AddService(db, &services[0])
	require.NoError(t, err)
	err = dbmodel.AddService(db, &services[1])
	require.NoError(t, err)

	// Run service detection again. The existing services should be returned.
	services = []dbmodel.Service{}
	for i := range app.Daemons {
		detected, err := DetectHAServices(db, app.Daemons[i])
		require.NoError(t, err)
		services = append(services, detected...)
	}
	require.Len(t, services, 2)

	// This is no longer a new service.
	require.False(t, services[0].IsNew())
	require.Equal(t, "dhcp4", services[0].HAService.HAType)
	require.Equal(t, "load-balancing", services[0].HAService.HAMode)
	require.Zero(t, services[0].HAService.PrimaryID)
	require.Equal(t, app.Daemons[0].ID, services[0].HAService.SecondaryID)
	require.Empty(t, services[0].HAService.BackupID)

	require.False(t, services[1].IsNew())
	require.Equal(t, "dhcp6", services[1].HAService.HAType)
	require.Equal(t, "hot-standby", services[1].HAService.HAMode)
	require.Zero(t, services[1].HAService.PrimaryID)
	require.Equal(t, app.Daemons[1].ID, services[1].HAService.SecondaryID)
	require.Empty(t, services[1].HAService.BackupID)

	// The daemons should be already associated with the services.
	require.Len(t, services[0].Daemons, 1)
	require.Len(t, services[1].Daemons, 1)

	// Add a backup server.
	dhcp4, err = dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)

	err = dhcp4.Configure(`{
		"Dhcp4": {
            "hooks-libraries": [
                {
                    "library": "libdhcp_ha.so",
                    "parameters": {
                        "high-availability": [{
                            "this-server-name": "server4",
                            "mode": "load-balancing",
                            "peers": [
								{
									"name": "server1",
									"url":  "http://192.0.2.33:8000",
									"role": "primary"
								},
								{
									"name": "server2",
									"url":  "http://192.0.2.66:8000",
									"role": "secondary"
								},
								{
									"name": "server4",
									"url":  "http://192.0.2.133:8000",
									"role": "backup"
								}
							]
                        }]
                    }
                }
            ]
		}
	}`)
	require.NoError(t, err)

	app, err = dhcp4.GetKea()
	require.NoError(t, err)

	// This time, we've added the server with only a DHCPv4 configuration.
	// Therefore, only a DHCPv4 service should be returned for this app.
	services, err = DetectHAServices(db, app.Daemons[0])
	require.NoError(t, err)
	require.Len(t, services, 1)
	require.False(t, services[0].IsNew())
	require.NotNil(t, services[0].HAService)
	require.Equal(t, "dhcp4", services[0].HAService.HAType)
	require.Equal(t, "load-balancing", services[0].HAService.HAMode)
	require.Zero(t, services[0].HAService.PrimaryID)
	require.NotZero(t, services[0].HAService.SecondaryID)
	require.Len(t, services[0].HAService.BackupID, 1)
	require.Equal(t, app.Daemons[0].ID, services[0].HAService.BackupID[0])

	require.Len(t, services[0].Daemons, 1)

	// We have to update the service in the database according to the value
	// returned by the detection routine.
	err = dbmodel.UpdateBaseHAService(db, services[0].HAService)
	require.NoError(t, err)

	// We also have to associate the daemon with the service.
	err = dbmodel.AddDaemonToService(db, services[0].ID, app.Daemons[0])
	require.NoError(t, err)

	// Add a primary server.
	kea, err = dbmodeltest.NewKea(db)
	require.NoError(t, err)

	dhcp4, err = kea.NewKeaDHCPv4Server()
	require.NoError(t, err)

	err = dhcp4.Configure(`{
		"Dhcp4": {
            "hooks-libraries": [
                {
                    "library": "libdhcp_ha.so",
                    "parameters": {
                        "high-availability": [{
                            "this-server-name": "server1",
                            "mode": "load-balancing",
                            "peers": [
								{
									"name": "server1",
									"url":  "http://192.0.2.33:8000",
									"role": "primary"
								},
								{
									"name": "server2",
									"url":  "http://192.0.2.66:8000",
									"role": "secondary"
								},
								{
									"name": "server4",
									"url":  "http://192.0.2.133:8000",
									"role": "backup"
								}
							]
                        }]
                    }
                }
            ]
		}
	}`)
	require.NoError(t, err)

	dhcp6, err = kea.NewKeaDHCPv6Server()
	require.NoError(t, err)

	err = dhcp6.Configure(`{
		"Dhcp6": {
            "hooks-libraries": [
                {
                    "library": "libdhcp_ha.so",
                    "parameters": {
                        "high-availability": [{
                            "this-server-name": "server1",
                            "mode": "hot-standby",
                            "peers": [
								{
									"name": "server1",
									"url":  "http://192.0.2.33:8000",
									"role": "primary"
								},
								{
									"name": "server3",
									"url":  "http://192.0.2.66:8000",
									"role": "standby"
								}
							]
                        }]
                    }
                }
            ]
		}
	}`)
	require.NoError(t, err)

	app, err = dhcp4.GetKea()
	require.NoError(t, err)

	// Since we have added two HA configurations for this app, there should
	// be two services returned, one for DHCPv4 and one for DHCPv6.
	services = []dbmodel.Service{}
	for i := range app.Daemons {
		detected, err := DetectHAServices(db, app.Daemons[i])
		require.NoError(t, err)
		services = append(services, detected...)
	}
	require.Len(t, services, 2)
	require.False(t, services[0].IsNew())
	require.NotNil(t, services[0].HAService)
	require.Equal(t, "dhcp4", services[0].HAService.HAType)
	require.Equal(t, "load-balancing", services[0].HAService.HAMode)
	require.Equal(t, app.Daemons[0].ID, services[0].HAService.PrimaryID)
	require.NotZero(t, services[0].HAService.SecondaryID)
	require.Len(t, services[0].HAService.BackupID, 1)

	require.False(t, services[1].IsNew())
	require.NotNil(t, services[1].HAService)
	require.Equal(t, "dhcp6", services[1].HAService.HAType)
	require.Equal(t, "hot-standby", services[1].HAService.HAMode)
	require.Equal(t, app.Daemons[1].ID, services[1].HAService.PrimaryID)
	require.NotZero(t, services[1].HAService.SecondaryID)
	require.Empty(t, services[1].HAService.BackupID)

	// The DHCPv4 service should have two daemons associated with it, one for
	// the secondary server and one for the backup. We will have to associate
	// the primary server with this service on our own.
	require.Len(t, services[0].Daemons, 2)

	// The DHCPv6 service should have only one daemon associated, i.e. the
	// secondary server, because we didn't add the backup server to it.
	require.Len(t, services[1].Daemons, 1)

	// Update the services in the database.
	err = dbmodel.UpdateBaseHAService(db, services[0].HAService)
	require.NoError(t, err)
	err = dbmodel.AddDaemonToService(db, services[0].ID, app.Daemons[0])
	require.NoError(t, err)

	err = dbmodel.UpdateBaseHAService(db, services[1].HAService)
	require.NoError(t, err)
	err = dbmodel.AddDaemonToService(db, services[1].ID, app.Daemons[1])
	require.NoError(t, err)

	// Add another DHCPv4 backup server.
	kea, err = dbmodeltest.NewKea(db)
	require.NoError(t, err)

	dhcp4, err = kea.NewKeaDHCPv4Server()
	require.NoError(t, err)

	err = dhcp4.Configure(`{
		"Dhcp4": {
            "hooks-libraries": [
                {
                    "library": "libdhcp_ha.so",
                    "parameters": {
                        "high-availability": [{
                            "this-server-name": "server5",
                            "mode": "load-balancing",
                            "peers": [
								{
									"name": "server1",
									"url":  "http://192.0.2.33:8000",
									"role": "primary"
								},
								{
									"name": "server2",
									"url":  "http://192.0.2.66:8000",
									"role": "secondary"
								},
								{
									"name": "server4",
									"url":  "http://192.0.2.133:8000",
									"role": "backup"
								},
								{
									"name": "server5",
									"url":  "http://192.0.2.166:8000",
									"role": "backup2"
								}
							]
                        }]
                    }
                }
            ]
		}
	}`)
	require.NoError(t, err)

	app, err = dhcp4.GetKea()
	require.NoError(t, err)

	services = []dbmodel.Service{}
	for i := range app.Daemons {
		detected, err := DetectHAServices(db, app.Daemons[i])
		require.NoError(t, err)
		services = append(services, detected...)
	}
	require.Len(t, services, 1)
	require.False(t, services[0].IsNew())
	require.NotNil(t, services[0].HAService)
	require.Equal(t, "dhcp4", services[0].HAService.HAType)
	require.Equal(t, "load-balancing", services[0].HAService.HAMode)
	require.NotZero(t, services[0].HAService.PrimaryID)
	require.NotZero(t, services[0].HAService.SecondaryID)
	require.Len(t, services[0].HAService.BackupID, 2)
	require.Contains(t, services[0].HAService.BackupID, app.Daemons[0].ID)

	require.Len(t, services[0].Daemons, 3)
}

// Test that a daemon doesn't belong to a blank service , i.e. a
// service that comprises no daemons.
func TestAppBelongsToHAServiceBlankService(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create blank service.
	service := &dbmodel.Service{
		BaseService: dbmodel.BaseService{
			ServiceType: "ha_dhcp",
		},
		HAService: &dbmodel.BaseHAService{
			HAType:       "dhcp4",
			Relationship: "server1",
		},
	}
	err := dbmodel.AddService(db, service)
	require.NoError(t, err)

	kea, err := dbmodeltest.NewKea(db)
	require.NoError(t, err)

	dhcp4, err := kea.NewKeaDHCPv4Server()
	require.NoError(t, err)

	err = dhcp4.Configure(`{
		"Dhcp4": {
            "hooks-libraries": [
                {
                    "library": "libdhcp_ha.so",
                    "parameters": {
                        "high-availability": [{
                            "this-server-name": "server1",
                            "mode": "load-balancing",
                            "peers": [
								{
									"name": "server1",
									"url":  "http://192.0.2.33:8000",
									"role": "primary"
								},
								{
									"name": "server2",
									"url":  "http://192.0.2.66:8000",
									"role": "secondary"
								},
								{
									"name": "server4",
									"url":  "http://192.0.2.133:8000",
									"role": "backup"
								}
							]
                        }]
                    }
                }
            ]
		}
	}`)
	require.NoError(t, err)

	app, err := dhcp4.GetKea()
	require.NoError(t, err)

	services, err := DetectHAServices(db, app.Daemons[0])
	require.NoError(t, err)
	require.Len(t, services, 1)
	require.True(t, services[0].IsNew())
}

// Test that a daemon can be dissociated with all services it belongs to.
func TestReduceHAServices(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fec := &storktest.FakeEventCenter{}
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()

	kea, err := dbmodeltest.NewKea(db)
	require.NoError(t, err)

	dhcp4, err := kea.NewKeaDHCPv4Server()
	require.NoError(t, err)

	err = dhcp4.Configure(`{
		"Dhcp4": {
            "hooks-libraries": [
                {
                    "library": "libdhcp_ha.so",
                    "parameters": {
                        "high-availability": [{
                            "this-server-name": "server1",
                            "mode": "load-balancing",
                            "peers": [
								{
									"name": "server1",
									"url":  "http://192.0.2.33:8000",
									"role": "primary"
								},
								{
									"name": "server2",
									"url":  "http://192.0.2.66:8000",
									"role": "secondary"
								},
								{
									"name": "server4",
									"url":  "http://192.0.2.133:8000",
									"role": "backup"
								}
							]
                        }]
                    }
                }
            ]
		}
	}`)
	require.NoError(t, err)

	keaApp, err := dhcp4.GetKea()
	require.NoError(t, err)

	// This call, apart from adding the app to the machine, will also associate the
	// app with the HA services.
	err = CommitAppIntoDB(db, keaApp, fec, nil, lookup)
	require.NoError(t, err)

	err = dhcp4.Configure(`{
		"Dhcp4": {
            "hooks-libraries": [
                {
                    "library": "libdhcp_ha.so",
                    "parameters": {
                        "high-availability": [{
                            "this-server-name": "server1",
                            "mode": "hot-standby",
                            "peers": [
								{
									"name": "server1",
									"url":  "http://192.0.2.33:8000",
									"role": "primary"
								},
								{
									"name": "server3",
									"url":  "http://192.0.2.66:8000",
									"role": "secondary"
								},
								{
									"name": "server4",
									"url":  "http://192.0.2.133:8000",
									"role": "backup"
								}
							]
                        }]
                    }
                }
            ]
		}
	}`)
	require.NoError(t, err)

	// Modify the HA configuration. It should replace the existing service with
	// a new one.
	keaApp, err = dhcp4.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, keaApp, fec, nil, lookup)
	require.NoError(t, err)

	// Make sure that new service has been created.
	services, err := dbmodel.GetDetailedAllServices(db)
	require.NoError(t, err)
	require.Len(t, services, 1)
	require.NotNil(t, services[0].HAService)
	require.Equal(t, "hot-standby", services[0].HAService.HAMode)
}

// Test that a hub server and multiple branch servers can be grouped
// into services.
func TestHubAndSpokeHAServices(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fec := &storktest.FakeEventCenter{}
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()

	// Create first branch server.
	kea, err := dbmodeltest.NewKea(db)
	require.NoError(t, err)

	dhcp4, err := kea.NewKeaDHCPv4Server()
	require.NoError(t, err)

	err = dhcp4.Configure(`{
		"Dhcp4": {
            "hooks-libraries": [
                {
                    "library": "libdhcp_ha.so",
                    "parameters": {
                        "high-availability": [{
                            "this-server-name": "server1",
                            "mode": "hot-standby",
                            "peers": [
								{
									"name": "server1",
									"url":  "http://192.0.2.33:8000",
									"role": "primary"
								},
								{
									"name": "server2",
									"url":  "http://192.0.2.66:8000",
									"role": "secondary"
								}
							]
                        }]
                    }
                }
            ]
		}
	}`)
	require.NoError(t, err)

	keaApp, err := dhcp4.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, keaApp, fec, nil, lookup)
	require.NoError(t, err)

	services, err := dbmodel.GetDetailedAllServices(db)
	require.NoError(t, err)
	require.Len(t, services, 1)
	require.NotNil(t, services[0].HAService)
	require.Equal(t, "hot-standby", services[0].HAService.HAMode)
	require.Len(t, services[0].Daemons, 1)
	require.EqualValues(t, keaApp.Daemons[0].ID, services[0].HAService.PrimaryID)

	// Create the hub
	kea, err = dbmodeltest.NewKea(db)
	require.NoError(t, err)

	dhcp4, err = kea.NewKeaDHCPv4Server()
	require.NoError(t, err)

	err = dhcp4.Configure(`{
			"Dhcp4": {
				"hooks-libraries": [
					{
						"library": "libdhcp_ha.so",
						"parameters": {
							"high-availability": [
								{
									"this-server-name": "server2",
									"mode": "hot-standby",
									"peers": [
										{
											"name": "server1",
											"url":  "http://192.0.2.33:8000",
											"role": "primary"
										},
										{
											"name": "server2",
											"url":  "http://192.0.2.66:8000",
											"role": "standby"
										}
									]
								},
								{
									"this-server-name": "server4",
									"mode": "hot-standby",
									"peers": [
										{
											"name": "server3",
											"url":  "http://192.0.2.99:8000",
											"role": "primary"
										},
										{
											"name": "server4",
											"url":  "http://192.0.2.133:8000",
											"role": "standby"
										}
									]
								}
							]
						}
					}
				]
			}
		}`)
	require.NoError(t, err)

	keaApp, err = dhcp4.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, keaApp, fec, nil, lookup)
	require.NoError(t, err)

	services, err = dbmodel.GetDetailedAllServices(db)
	require.NoError(t, err)
	require.Len(t, services, 2)
	require.NotNil(t, services[0].HAService)
	require.Equal(t, "hot-standby", services[0].HAService.HAMode)
	require.Len(t, services[0].Daemons, 2)
	require.EqualValues(t, keaApp.Daemons[0].ID, services[0].HAService.SecondaryID)
	require.Equal(t, "hot-standby", services[1].HAService.HAMode)
	require.Len(t, services[1].Daemons, 1)
	require.EqualValues(t, keaApp.Daemons[0].ID, services[1].HAService.SecondaryID)

	// Create another branch server.
	kea, err = dbmodeltest.NewKea(db)
	require.NoError(t, err)

	dhcp4, err = kea.NewKeaDHCPv4Server()
	require.NoError(t, err)

	err = dhcp4.Configure(`{
			"Dhcp4": {
				"hooks-libraries": [
					{
						"library": "libdhcp_ha.so",
						"parameters": {
							"high-availability": [
								{
									"this-server-name": "server3",
									"mode": "hot-standby",
									"peers": [
										{
											"name": "server3",
											"url":  "http://192.0.2.99:8000",
											"role": "primary"
										},
										{
											"name": "server4",
											"url":  "http://192.0.2.133:8000",
											"role": "standby"
										}
									]
								}
							]
						}
					}
				]
			}
		}`)
	require.NoError(t, err)

	keaApp, err = dhcp4.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, keaApp, fec, nil, lookup)
	require.NoError(t, err)

	services, err = dbmodel.GetDetailedAllServices(db)
	require.NoError(t, err)
	require.Len(t, services, 2)
	require.NotNil(t, services[0].HAService)
	require.Equal(t, "hot-standby", services[0].HAService.HAMode)
	require.Len(t, services[0].Daemons, 2)
	require.Equal(t, "hot-standby", services[1].HAService.HAMode)
	require.Len(t, services[1].Daemons, 2)
	require.EqualValues(t, keaApp.Daemons[0].ID, services[1].HAService.PrimaryID)
}

// Test error cases while detecting the HA services.
func TestDetectHAServicesErrors(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fec := &storktest.FakeEventCenter{}
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()

	kea, err := dbmodeltest.NewKea(db)
	require.NoError(t, err)

	dhcp4, err := kea.NewKeaDHCPv4Server()
	require.NoError(t, err)

	err = dhcp4.Configure(`{
		"Dhcp4": {
            "hooks-libraries": [
                {
                    "library": "libdhcp_ha.so",
                    "parameters": {
                        "high-availability": [{
                            "this-server-name": "server1",
                            "mode": "hot-standby",
                            "peers": [
								{
									"name": "server1",
									"url":  "http://192.0.2.33:8000",
									"role": "primary"
								},
								{
									"name": "server2",
									"url":  "http://192.0.2.66:8000",
									"role": "standby"
								}
							]
                        }]
                    }
                }
            ]
		}
	}`)
	require.NoError(t, err)

	keaApp, err := dhcp4.GetKea()
	require.NoError(t, err)

	err = CommitAppIntoDB(db, keaApp, fec, nil, lookup)
	require.NoError(t, err)

	services, err := dbmodel.GetDetailedAllServices(db)
	require.NoError(t, err)
	require.Len(t, services, 1)

	t.Run("not a Kea daemon", func(t *testing.T) {
		daemon := &dbmodel.Daemon{}
		services, err := DetectHAServices(db, daemon)
		require.NoError(t, err)
		require.Empty(t, services)
	})

	t.Run("no HA hook library", func(t *testing.T) {
		config, err := dbmodel.NewKeaConfigFromJSON(`{
			"Dhcp4": {}
		}`)
		require.NoError(t, err)
		daemon := &dbmodel.Daemon{
			KeaDaemon: &dbmodel.KeaDaemon{
				Config:        config,
				KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
			},
		}

		services, err := DetectHAServices(db, daemon)
		require.NoError(t, err)
		require.Empty(t, services)
	})

	t.Run("invalid HA configuration", func(t *testing.T) {
		config, err := dbmodel.NewKeaConfigFromJSON(`{
			"Dhcp4": {
				"hooks-libraries": [
					{
						"library": "libdhcp_ha.so",
						"parameters": {
							"high-availability": [{
								"this-server-name": "server2",
								"mode": "hot-standby",
								"peers": [
									{
										"name": "server1",
										"role": "primary"
									},
									{
										"name": "server2",
										"role": "standby"
									}
								]
							}]
						}
					}
				]
			}
		}`)
		require.NoError(t, err)
		daemon := &dbmodel.Daemon{
			KeaDaemon: &dbmodel.KeaDaemon{
				Config:        config,
				KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
			},
		}
		services, err := DetectHAServices(db, daemon)
		require.Error(t, err)
		require.Empty(t, services)
	})

	t.Run("no matching peer", func(t *testing.T) {
		config, err := dbmodel.NewKeaConfigFromJSON(`{
			"Dhcp4": {
				"hooks-libraries": [
					{
						"library": "libdhcp_ha.so",
						"parameters": {
							"high-availability": [{
								"this-server-name": "server3",
								"mode": "hot-standby",
								"peers": [
									{
										"name": "server1",
										"url":  "http://192.0.2.33:8000",
										"role": "primary"
									},
									{
										"name": "server2",
										"url":  "http://192.0.2.66:8000",
										"role": "standby"
									}
								]
							}]
						}
					}
				]
			}
		}`)
		require.NoError(t, err)
		daemon := &dbmodel.Daemon{
			KeaDaemon: &dbmodel.KeaDaemon{
				Config:        config,
				KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
			},
		}
		services, err := DetectHAServices(db, daemon)
		require.Error(t, err)
		require.Empty(t, services)
	})
}
