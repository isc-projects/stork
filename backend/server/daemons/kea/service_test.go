package kea

import (
	"testing"

	require "github.com/stretchr/testify/require"
	"isc.org/stork/datamodel/daemonname"
	dbmodel "isc.org/stork/server/database/model"
	dbmodeltest "isc.org/stork/server/database/model/test"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test/dbmodel"
)

// Multi step test verifying that services can be gradually created from the
// Kea daemons being added to the database.
func TestDetectHAServices(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	dhcp4, err := dbmodeltest.NewKeaDHCPv4Server(db)
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

	dhcp6, err := dbmodeltest.NewKeaDHCPv6Server(db)
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

	daemon4, err := dhcp4.GetDaemon()
	require.NoError(t, err)

	daemon6, err := dhcp6.GetDaemon()
	require.NoError(t, err)

	// Run service detection for all daemons.
	var services []dbmodel.Service
	detected, err := DetectHAServices(db, daemon4)
	require.NoError(t, err)
	services = append(services, detected...)

	detected, err = DetectHAServices(db, daemon6)
	require.NoError(t, err)
	services = append(services, detected...)

	// There should be two services returned, one for DHCPv4 and one for DHCPv6.
	require.Len(t, services, 2)

	// Check the DHCPv4 service first.
	require.True(t, services[0].IsNew())
	require.NotNil(t, services[0].HAService)
	require.Equal(t, daemonname.DHCPv4, services[0].HAService.HAType)
	require.Equal(t, "load-balancing", services[0].HAService.HAMode)
	require.Equal(t, "server2", services[0].HAService.Relationship)
	require.Zero(t, services[0].HAService.PrimaryID)
	require.Equal(t, daemon4.ID, services[0].HAService.SecondaryID)
	require.Empty(t, services[0].HAService.BackupID)
	require.True(t, services[0].HAService.PrimaryStatusCollectedAt.IsZero())
	require.True(t, services[0].HAService.SecondaryStatusCollectedAt.IsZero())
	require.Empty(t, services[0].HAService.PrimaryLastState)
	require.Empty(t, services[0].HAService.SecondaryLastState)

	// Check the DHCPv6 service.
	require.True(t, services[1].IsNew())
	require.NotNil(t, services[1].HAService)
	require.Equal(t, daemonname.DHCPv6, services[1].HAService.HAType)
	require.Equal(t, "hot-standby", services[1].HAService.HAMode)
	require.Equal(t, "server2", services[0].HAService.Relationship)
	require.Zero(t, services[1].HAService.PrimaryID)
	require.Equal(t, daemon6.ID, services[1].HAService.SecondaryID)
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
	detected, err = DetectHAServices(db, daemon4)
	require.NoError(t, err)
	services = append(services, detected...)

	detected, err = DetectHAServices(db, daemon6)
	require.NoError(t, err)
	services = append(services, detected...)
	require.Len(t, services, 2)

	// This is no longer a new service.
	require.False(t, services[0].IsNew())
	require.Equal(t, daemonname.DHCPv4, services[0].HAService.HAType)
	require.Equal(t, "load-balancing", services[0].HAService.HAMode)
	require.Zero(t, services[0].HAService.PrimaryID)
	require.Equal(t, daemon4.ID, services[0].HAService.SecondaryID)
	require.Empty(t, services[0].HAService.BackupID)

	require.False(t, services[1].IsNew())
	require.Equal(t, daemonname.DHCPv6, services[1].HAService.HAType)
	require.Equal(t, "hot-standby", services[1].HAService.HAMode)
	require.Zero(t, services[1].HAService.PrimaryID)
	require.Equal(t, daemon6.ID, services[1].HAService.SecondaryID)
	require.Empty(t, services[1].HAService.BackupID)

	// The daemons should be already associated with the services.
	require.Len(t, services[0].Daemons, 1)
	require.Len(t, services[1].Daemons, 1)

	// Add a backup server.
	dhcp4Backup, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)

	err = dhcp4Backup.Configure(`{
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

	daemonBackup, err := dhcp4Backup.GetDaemon()
	require.NoError(t, err)

	// This time, we've added the server with only a DHCPv4 configuration.
	// Therefore, only a DHCPv4 service should be returned for this daemon.
	services, err = DetectHAServices(db, daemonBackup)
	require.NoError(t, err)
	require.Len(t, services, 1)
	require.False(t, services[0].IsNew())
	require.NotNil(t, services[0].HAService)
	require.Equal(t, daemonname.DHCPv4, services[0].HAService.HAType)
	require.Equal(t, "load-balancing", services[0].HAService.HAMode)
	require.Zero(t, services[0].HAService.PrimaryID)
	require.NotZero(t, services[0].HAService.SecondaryID)
	require.Len(t, services[0].HAService.BackupID, 1)
	require.Equal(t, daemonBackup.ID, services[0].HAService.BackupID[0])

	require.Len(t, services[0].Daemons, 1)

	// We have to update the service in the database according to the value
	// returned by the detection routine.
	err = dbmodel.UpdateBaseHAService(db, services[0].HAService)
	require.NoError(t, err)

	// We also have to associate the daemon with the service.
	err = dbmodel.AddDaemonToService(db, services[0].ID, daemonBackup)
	require.NoError(t, err)

	// Add a primary server.
	dhcp4Primary, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)

	err = dhcp4Primary.Configure(`{
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

	dhcp6Primary, err := dbmodeltest.NewKeaDHCPv6Server(db)
	require.NoError(t, err)

	err = dhcp6Primary.Configure(`{
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

	daemonPrimary4, err := dhcp4Primary.GetDaemon()
	require.NoError(t, err)

	daemonPrimary6, err := dhcp6Primary.GetDaemon()
	require.NoError(t, err)

	// Since we have added two HA configurations for these daemons, there should
	// be two services returned, one for DHCPv4 and one for DHCPv6.
	services = []dbmodel.Service{}
	detected, err = DetectHAServices(db, daemonPrimary4)
	require.NoError(t, err)
	services = append(services, detected...)

	detected, err = DetectHAServices(db, daemonPrimary6)
	require.NoError(t, err)
	services = append(services, detected...)
	require.Len(t, services, 2)
	require.False(t, services[0].IsNew())
	require.NotNil(t, services[0].HAService)
	require.Equal(t, daemonname.DHCPv4, services[0].HAService.HAType)
	require.Equal(t, "load-balancing", services[0].HAService.HAMode)
	require.Equal(t, daemonPrimary4.ID, services[0].HAService.PrimaryID)
	require.NotZero(t, services[0].HAService.SecondaryID)
	require.Len(t, services[0].HAService.BackupID, 1)

	require.False(t, services[1].IsNew())
	require.NotNil(t, services[1].HAService)
	require.Equal(t, daemonname.DHCPv6, services[1].HAService.HAType)
	require.Equal(t, "hot-standby", services[1].HAService.HAMode)
	require.Equal(t, daemonPrimary6.ID, services[1].HAService.PrimaryID)
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
	err = dbmodel.AddDaemonToService(db, services[0].ID, daemonPrimary4)
	require.NoError(t, err)

	err = dbmodel.UpdateBaseHAService(db, services[1].HAService)
	require.NoError(t, err)
	err = dbmodel.AddDaemonToService(db, services[1].ID, daemonPrimary6)
	require.NoError(t, err)

	// Add another DHCPv4 backup server.
	dhcp4AnotherBackup, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)

	err = dhcp4AnotherBackup.Configure(`{
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

	daemonAnotherBackup, err := dhcp4AnotherBackup.GetDaemon()
	require.NoError(t, err)

	// The services shouldn't be changed when they are detected again.
	for i := 0; i < 5; i++ {
		services = []dbmodel.Service{}
		detected, err := DetectHAServices(db, daemonAnotherBackup)
		require.NoError(t, err)
		for _, d := range detected {
			err = dbmodel.UpdateBaseHAService(db, d.HAService)
			require.NoError(t, err)
		}
		services = append(services, detected...)
		require.Len(t, services, 1)
		require.False(t, services[0].IsNew())
		require.NotNil(t, services[0].HAService)
		require.Equal(t, daemonname.DHCPv4, services[0].HAService.HAType)
		require.Equal(t, "load-balancing", services[0].HAService.HAMode)
		require.NotZero(t, services[0].HAService.PrimaryID)
		require.NotZero(t, services[0].HAService.SecondaryID)
		require.Len(t, services[0].HAService.BackupID, 2)
		require.Contains(t, services[0].HAService.BackupID, daemonAnotherBackup.ID)

		require.Len(t, services[0].Daemons, 3)
	}
}

// Test that a daemon doesn't belong to a blank service , i.e. a
// service that comprises no daemons.
func TestAppBelongsToHAServiceBlankService(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create blank service.
	service := &dbmodel.Service{
		BaseService: dbmodel.BaseService{},
		HAService: &dbmodel.BaseHAService{
			HAType:       "dhcp4",
			Relationship: "server1",
		},
	}
	err := dbmodel.AddService(db, service)
	require.NoError(t, err)

	dhcp4, err := dbmodeltest.NewKeaDHCPv4Server(db)
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

	daemon, err := dhcp4.GetDaemon()
	require.NoError(t, err)

	services, err := DetectHAServices(db, daemon)
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

	dhcp4, err := dbmodeltest.NewKeaDHCPv4Server(db)
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

	daemon, err := dhcp4.GetDaemon()
	require.NoError(t, err)

	// This call, apart from adding the daemon to the machine, will also associate the
	// daemon with the HA services.
	err = CommitDaemonsIntoDB(db,
		[]*dbmodel.Daemon{daemon},
		fec,
		[]DaemonStateMeta{{IsConfigChanged: true}},
		lookup,
	)
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
	daemon, err = dhcp4.GetDaemon()
	require.NoError(t, err)

	err = CommitDaemonsIntoDB(db,
		[]*dbmodel.Daemon{daemon},
		fec,
		[]DaemonStateMeta{{IsConfigChanged: true}},
		lookup,
	)
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
	dhcp4, err := dbmodeltest.NewKeaDHCPv4Server(db)
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

	daemon, err := dhcp4.GetDaemon()
	require.NoError(t, err)

	err = CommitDaemonsIntoDB(db,
		[]*dbmodel.Daemon{daemon},
		fec,
		[]DaemonStateMeta{{IsConfigChanged: true}},
		lookup,
	)
	require.NoError(t, err)

	services, err := dbmodel.GetDetailedAllServices(db)
	require.NoError(t, err)
	require.Len(t, services, 1)
	require.NotNil(t, services[0].HAService)
	require.Equal(t, "hot-standby", services[0].HAService.HAMode)
	require.Len(t, services[0].Daemons, 1)
	require.EqualValues(t, daemon.ID, services[0].HAService.PrimaryID)

	// Create the hub
	dhcp4Hub, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)

	err = dhcp4Hub.Configure(`{
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

	daemonHub, err := dhcp4Hub.GetDaemon()
	require.NoError(t, err)

	err = CommitDaemonsIntoDB(db,
		[]*dbmodel.Daemon{daemonHub},
		fec,
		[]DaemonStateMeta{{IsConfigChanged: true}},
		lookup,
	)
	require.NoError(t, err)

	services, err = dbmodel.GetDetailedAllServices(db)
	require.NoError(t, err)
	require.Len(t, services, 2)
	require.NotNil(t, services[0].HAService)
	require.Equal(t, "hot-standby", services[0].HAService.HAMode)
	require.Len(t, services[0].Daemons, 2)
	require.EqualValues(t, daemonHub.ID, services[0].HAService.SecondaryID)
	require.Equal(t, "hot-standby", services[1].HAService.HAMode)
	require.Len(t, services[1].Daemons, 1)
	require.EqualValues(t, daemonHub.ID, services[1].HAService.SecondaryID)

	// Create another branch server.
	dhcp4Branch, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)

	err = dhcp4Branch.Configure(`{
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

	daemonBranch, err := dhcp4Branch.GetDaemon()
	require.NoError(t, err)

	err = CommitDaemonsIntoDB(db,
		[]*dbmodel.Daemon{daemonBranch},
		fec,
		[]DaemonStateMeta{{IsConfigChanged: true}},
		lookup,
	)
	require.NoError(t, err)

	services, err = dbmodel.GetDetailedAllServices(db)
	require.NoError(t, err)
	require.Len(t, services, 2)
	require.NotNil(t, services[0].HAService)
	require.Equal(t, "hot-standby", services[0].HAService.HAMode)
	require.Len(t, services[0].Daemons, 2)
	require.Equal(t, "hot-standby", services[1].HAService.HAMode)
	require.Len(t, services[1].Daemons, 2)
	require.EqualValues(t, daemonBranch.ID, services[1].HAService.PrimaryID)
}

// Test error cases while detecting the HA services.
func TestDetectHAServicesErrors(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fec := &storktest.FakeEventCenter{}
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()

	dhcp4, err := dbmodeltest.NewKeaDHCPv4Server(db)
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

	daemon, err := dhcp4.GetDaemon()
	require.NoError(t, err)

	err = CommitDaemonsIntoDB(db,
		[]*dbmodel.Daemon{daemon},
		fec,
		[]DaemonStateMeta{{IsConfigChanged: true}},
		lookup,
	)
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
		dhcp4NoHA, err := dbmodeltest.NewKeaDHCPv4Server(db)
		require.NoError(t, err)

		err = dhcp4NoHA.Configure(`{
			"Dhcp4": {}
		}`)
		require.NoError(t, err)

		daemonNoHA, err := dhcp4NoHA.GetDaemon()
		require.NoError(t, err)

		services, err := DetectHAServices(db, daemonNoHA)
		require.NoError(t, err)
		require.Empty(t, services)
	})

	t.Run("invalid HA configuration", func(t *testing.T) {
		dhcp4Invalid, err := dbmodeltest.NewKeaDHCPv4Server(db)
		require.NoError(t, err)

		err = dhcp4Invalid.Configure(`{
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

		daemonInvalid, err := dhcp4Invalid.GetDaemon()
		require.NoError(t, err)

		services, err := DetectHAServices(db, daemonInvalid)
		require.Error(t, err)
		require.Empty(t, services)
	})

	t.Run("no matching peer", func(t *testing.T) {
		dhcp4NoMatch, err := dbmodeltest.NewKeaDHCPv4Server(db)
		require.NoError(t, err)

		err = dhcp4NoMatch.Configure(`{
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

		daemonNoMatch, err := dhcp4NoMatch.GetDaemon()
		require.NoError(t, err)

		services, err := DetectHAServices(db, daemonNoMatch)
		require.Error(t, err)
		require.Empty(t, services)
	})
}
