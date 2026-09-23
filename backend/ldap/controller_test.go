package ldap_test

import (
	"crypto/tls"
	"errors"
	"testing"
	"time"

	goldap "github.com/go-ldap/ldap/v3"
	"github.com/stretchr/testify/require"
	"isc.org/stork/ldap"
	"isc.org/stork/server/authdata"
)

// Base error for the testing purposes.
var errTest = errors.New("test error")

// Tests if the LDAP controller can be created.
func TestNewLDAPController(t *testing.T) {
	// Arrange & Act
	controller := ldap.NewLDAPController(ldap.Settings{}, newMockDriver())

	// Assert
	require.NotNil(t, controller)
}

// Tests if the LDAP controller is not configured with TLS if LDAP server is
// served over unsecure protocol.
func TestConnectWithoutTLS(t *testing.T) {
	// Arrange
	mock := newMockDriver()
	settings := ldap.Settings{
		DialURL: "ldap://foobar:42",
		Timeout: 24 * time.Hour,
	}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	err := controller.Connect()

	// Assert
	require.NoError(t, err)
	require.Len(t, mock.calls, 1)
	require.Equal(t, "Dial", mock.calls[0].name)
	require.Equal(t, "ldap://foobar:42", mock.calls[0].arguments["url"])
	require.Equal(t, 24*time.Hour, mock.calls[0].arguments["timeout"])
	require.Nil(t, mock.calls[0].arguments["tls"])
}

// Tests that the LDAP controller returns an error if the connection cannot be
// established.
func TestConnectWithError(t *testing.T) {
	// Arrange
	mock := newMockDriver()
	mock.returnDial = errTest
	settings := ldap.Settings{
		DialURL: "ldap://foobar:42",
		Timeout: 24 * time.Hour,
	}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	err := controller.Connect()

	// Assert
	require.Error(t, err)
	require.Len(t, mock.calls, 1)
	require.Equal(t, "Dial", mock.calls[0].name)
	require.Equal(t, "ldap://foobar:42", mock.calls[0].arguments["url"])
	require.Equal(t, 24*time.Hour, mock.calls[0].arguments["timeout"])
	require.Nil(t, mock.calls[0].arguments["tls"])
}

// Tests that the LDAP controller can be configured to skip TLS server
// verification.
func TestConnectWithTLSSkipVerification(t *testing.T) {
	// Arrange
	mock := newMockDriver()
	settings := ldap.Settings{
		DialURL:                   "ldaps://foobar:42",
		Timeout:                   24 * time.Hour,
		TLSSkipServerVerification: true,
	}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	err := controller.Connect()

	// Assert
	require.NoError(t, err)
	require.Len(t, mock.calls, 1)
	require.Equal(t, "Dial", mock.calls[0].name)
	require.Equal(t, "ldaps://foobar:42", mock.calls[0].arguments["url"])
	require.Equal(t, 24*time.Hour, mock.calls[0].arguments["timeout"])
	require.NotNil(t, mock.calls[0].arguments["tls"])
	require.True(t, mock.calls[0].arguments["tls"].(*tls.Config).InsecureSkipVerify)
}

// Tests that the LDAP controller is configured with TLS if LDAP server is
// served over secure protocol.
func TestConnectWithTLS(t *testing.T) {
	// Arrange
	mock := newMockDriver()
	settings := ldap.Settings{
		DialURL:                   "ldaps://foobar:42",
		Timeout:                   24 * time.Hour,
		TLSSkipServerVerification: false,
	}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	err := controller.Connect()

	// Assert
	require.NoError(t, err)
	require.Len(t, mock.calls, 1)
	require.Equal(t, "Dial", mock.calls[0].name)
	require.Equal(t, "ldaps://foobar:42", mock.calls[0].arguments["url"])
	require.Equal(t, 24*time.Hour, mock.calls[0].arguments["timeout"])
	require.NotNil(t, mock.calls[0].arguments["tls"])
	require.False(t, mock.calls[0].arguments["tls"].(*tls.Config).InsecureSkipVerify)
}

// Tests that the LDAP controller binds the maintenance user with the proper
// credentials.
func TestBindAsMaintenanceUser(t *testing.T) {
	mock := newMockDriver()
	settings := ldap.Settings{
		Root:         "foo",
		BindUserDN:   "cn=bar",
		BindPassword: "boz",
	}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	err := controller.BindAsMaintenanceUser()

	// Assert
	require.NoError(t, err)
	require.Len(t, mock.calls, 1)
	require.Equal(t, "SimpleBind", mock.calls[0].name)
	require.Equal(t, "cn=bar", mock.calls[0].arguments["userDN"])
	require.Equal(t, "boz", mock.calls[0].arguments["password"])
	require.False(t, mock.calls[0].arguments["allowEmptyPassword"].(bool))
}

// Tests that the maintenance user can be bound with an empty password.
func TestBindAsMaintenanceUserWithEmptyPassword(t *testing.T) {
	mock := newMockDriver()
	settings := ldap.Settings{
		Root:         "foo",
		BindUserDN:   "cn=bar",
		BindPassword: "",
	}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	err := controller.BindAsMaintenanceUser()

	// Assert
	require.NoError(t, err)
	require.Len(t, mock.calls, 1)
	require.Equal(t, "SimpleBind", mock.calls[0].name)
	require.Equal(t, "cn=bar", mock.calls[0].arguments["userDN"])
	require.Empty(t, mock.calls[0].arguments["password"])
	require.True(t, mock.calls[0].arguments["allowEmptyPassword"].(bool))
}

// Tests that the binding of the maintenance user fails if the connection to
// the LDAP server cannot be established.
func TestBindAsMaintenanceUserWithError(t *testing.T) {
	// Arrange
	mock := newMockDriver()
	mock.returnSimpleBind = errTest
	settings := ldap.Settings{}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	err := controller.BindAsMaintenanceUser()

	// Assert
	require.Error(t, err)
}

// Tests that the LDAP controller can search for user DN by username.
func TestSearchForUserDN(t *testing.T) {
	// Arrange
	mock := newMockDriver()
	mock.returnSearchValue = &goldap.SearchResult{
		Entries: []*goldap.Entry{
			{DN: "foo"},
		},
	}
	settings := ldap.Settings{
		Root: "root",
		AttributeNames: ldap.LDAPAttributeNames{
			ObjectClassUser: "objectClassUser",
			UserID:          "userID",
		},
	}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	userDN, err := controller.SearchForUserDN("bar")

	// Assert
	require.NoError(t, err)
	require.Equal(t, "foo", userDN)
	require.Len(t, mock.calls, 1)
	require.Equal(t, "Search", mock.calls[0].name)
	request, ok := mock.calls[0].arguments["request"].(*goldap.SearchRequest)
	require.True(t, ok)
	require.NotNil(t, request)
	require.Equal(t, "root", request.BaseDN)
	require.Equal(t, goldap.ScopeWholeSubtree, request.Scope)
	require.Equal(t, goldap.NeverDerefAliases, request.DerefAliases)
	require.Zero(t, request.SizeLimit)
	require.Zero(t, request.TimeLimit)
	require.False(t, request.TypesOnly)
	require.Equal(t, "(&(objectClass=objectClassUser)(userID=bar))", request.Filter)
	require.Len(t, request.Attributes, 1)
	require.Contains(t, request.Attributes, "dn")
}

// Tests that an error is returned if there are no DN for a given username.
func TestSearchForUserDNErrorIfNoEntries(t *testing.T) {
	// Arrange
	mock := newMockDriver()
	mock.returnSearchValue = &goldap.SearchResult{
		Entries: []*goldap.Entry{},
	}
	settings := ldap.Settings{}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	userDN, err := controller.SearchForUserDN("bar")

	// Assert
	require.ErrorContains(t, err, "searching for user DN failed")
	require.ErrorContains(t, err, "no data found")
	require.Empty(t, userDN)
}

// Tests that an error is returned if the searching for DN returns too many
// entries.
func TestSearchForUserDNErrorIfTooManyEntries(t *testing.T) {
	// Arrange
	mock := newMockDriver()
	mock.returnSearchValue = &goldap.SearchResult{
		Entries: []*goldap.Entry{
			{DN: "foo"},
			{DN: "boz"},
		},
	}
	settings := ldap.Settings{}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	userDN, err := controller.SearchForUserDN("bar")

	// Assert
	require.ErrorContains(t, err, "too many user entries")
	require.Empty(t, userDN)
}

// Tests that the LDAP controller can search for user profile by username.
func TestSearchForUserProfile(t *testing.T) {
	// Arrange
	mock := newMockDriver()
	mock.returnSearchValue = &goldap.SearchResult{
		Entries: []*goldap.Entry{
			{
				DN: "cn=foobar,dc=example,dc=com",
				Attributes: []*goldap.EntryAttribute{
					{
						Name:   "uuid",
						Values: []string{"0000-0000-0000-0000"},
					},
					{
						Name:   "firstName",
						Values: []string{"foo"},
					},
					{
						Name:   "lastName",
						Values: []string{"bar"},
					},
					{
						Name:   "email",
						Values: []string{"foo@bar.com"},
					},
				},
			},
		},
	}
	settings := ldap.Settings{
		Root: "root",
		AttributeNames: ldap.LDAPAttributeNames{
			ObjectClassUser:  "objectClassUser",
			UserID:           "userID",
			FirstName:        "firstName",
			LastName:         "lastName",
			Email:            "email",
			UniqueIdentifier: "uuid",
		},
	}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	userProfile, err := controller.SearchForUserProfile("username")

	// Assert
	require.NoError(t, err)
	require.NotNil(t, userProfile)
	require.Equal(t, "0000-0000-0000-0000", userProfile.ID)
	require.Equal(t, "foo", userProfile.Name)
	require.Equal(t, "bar", userProfile.Lastname)
	require.Equal(t, "foo@bar.com", userProfile.Email)
	require.Equal(t, "username", userProfile.Login)
	require.Nil(t, userProfile.Groups)
}

// Tests that the LDAP controller can search for user profile by username and
// it can handle DN as unique identifier.
func TestSearchForUserProfileWithDNAsUniqueIdentifier(t *testing.T) {
	// Arrange
	mock := newMockDriver()
	mock.returnSearchValue = &goldap.SearchResult{
		Entries: []*goldap.Entry{
			{
				DN: "cn=foobar,dc=example,dc=com",
				Attributes: []*goldap.EntryAttribute{
					{
						Name:   "uuid",
						Values: []string{"0000-0000-0000-0000"},
					},
					{
						Name:   "firstName",
						Values: []string{"foo"},
					},
					{
						Name:   "lastName",
						Values: []string{"bar"},
					},
					{
						Name:   "email",
						Values: []string{"foo@bar.com"},
					},
				},
			},
		},
	}
	settings := ldap.Settings{
		Root: "root",
		AttributeNames: ldap.LDAPAttributeNames{
			ObjectClassUser:  "objectClassUser",
			UserID:           "userID",
			FirstName:        "firstName",
			LastName:         "lastName",
			Email:            "email",
			UniqueIdentifier: "dn",
		},
	}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	userProfile, err := controller.SearchForUserProfile("username")

	// Assert
	require.NoError(t, err)
	require.NotNil(t, userProfile)
	require.Equal(t, "cn=foobar,dc=example,dc=com", userProfile.ID)
	require.Equal(t, "foo", userProfile.Name)
	require.Equal(t, "bar", userProfile.Lastname)
	require.Equal(t, "foo@bar.com", userProfile.Email)
	require.Equal(t, "username", userProfile.Login)
	require.Nil(t, userProfile.Groups)
}

// Tests that the user profile can be found even if it misses optional
// attributes. The optional and mandatory attributes are specified by the
// object class in the LDAP server configuration.
func TestSearchForUserProfileMissingOptionalAttributes(t *testing.T) {
	// Arrange
	mock := newMockDriver()
	mock.returnSearchValue = &goldap.SearchResult{
		Entries: []*goldap.Entry{
			{
				DN: "cn=foobar,dc=example,dc=com",
				Attributes: []*goldap.EntryAttribute{{
					// This attribute is mandatory.
					Name:   "uuid",
					Values: []string{"0000-0000-0000-0000"},
				}},
			},
		},
	}
	settings := ldap.Settings{
		Root: "root",
		AttributeNames: ldap.LDAPAttributeNames{
			ObjectClassUser:  "objectClassUser",
			UserID:           "userID",
			FirstName:        "firstName",
			LastName:         "lastName",
			Email:            "email",
			UniqueIdentifier: "uuid",
		},
	}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	userProfile, err := controller.SearchForUserProfile("username")

	// Assert
	require.NoError(t, err)
	require.NotNil(t, userProfile)
	require.Equal(t, "0000-0000-0000-0000", userProfile.ID)
	require.Empty(t, userProfile.Name)
	require.Empty(t, userProfile.Lastname)
	require.Empty(t, userProfile.Email)
	require.Equal(t, "username", userProfile.Login)
	require.Nil(t, userProfile.Groups)
}

// Tests that the user profile cannot be found if it misses mandatory
// attributes.
func TestSearchForUserProfileMissingMandatoryAttributes(t *testing.T) {
	// Arrange
	mock := newMockDriver()
	mock.returnSearchValue = &goldap.SearchResult{
		Entries: []*goldap.Entry{
			{
				DN:         "cn=foobar,dc=example,dc=com",
				Attributes: []*goldap.EntryAttribute{},
			},
		},
	}
	settings := ldap.Settings{
		Root: "root",
		AttributeNames: ldap.LDAPAttributeNames{
			ObjectClassUser:  "objectClassUser",
			UserID:           "userID",
			FirstName:        "firstName",
			LastName:         "lastName",
			Email:            "email",
			UniqueIdentifier: "uuid",
		},
	}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	userProfile, err := controller.SearchForUserProfile("username")

	// Assert
	require.ErrorContains(t, err, "missing unique identifier attribute (uuid)")
	require.Nil(t, userProfile)
}

// Tests that an error is returned if there is no user profile for a given
// username.
func TestSearchForUserProfileMissingProfile(t *testing.T) {
	// Arrange
	mock := newMockDriver()
	mock.returnSearchValue = &goldap.SearchResult{
		Entries: []*goldap.Entry{},
	}
	settings := ldap.Settings{}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	userProfile, err := controller.SearchForUserProfile("username")

	// Assert
	require.ErrorContains(t, err, "searching for user profile failed")
	require.ErrorContains(t, err, "no data found")
	require.Nil(t, userProfile)
}

// Tests that an error is returned if the searching for user profile fails.
func TestSearchForUserProfileError(t *testing.T) {
	// Arrange
	mock := newMockDriver()
	mock.returnSearchErr = errTest
	settings := ldap.Settings{}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	userProfile, err := controller.SearchForUserProfile("username")

	// Assert
	require.ErrorContains(t, err, "test error")
	require.Nil(t, userProfile)
}

// Tests that the LDAP controller can bind as an arbitrary user.
func TestBindAsUser(t *testing.T) {
	// Arrange
	mock := newMockDriver()
	settings := ldap.Settings{}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	err := controller.BindAsUser("foo", "bar")

	// Assert
	require.NoError(t, err)
	require.Len(t, mock.calls, 1)
	require.Equal(t, "SimpleBind", mock.calls[0].name)
	require.Equal(t, "foo", mock.calls[0].arguments["userDN"])
	require.Equal(t, "bar", mock.calls[0].arguments["password"])
	require.False(t, mock.calls[0].arguments["allowEmptyPassword"].(bool))
}

// Tests that the LDAP controller can bind as an arbitrary user with an empty
// password.
func TestBindAsUserWithEmptyPassword(t *testing.T) {
	mock := newMockDriver()
	settings := ldap.Settings{}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	err := controller.BindAsUser("foo", "")

	// Assert
	// There is an error in runtime because empty password is not allowed.
	require.NoError(t, err)
	require.Len(t, mock.calls, 1)
	require.Equal(t, "SimpleBind", mock.calls[0].name)
	require.Equal(t, "foo", mock.calls[0].arguments["userDN"])
	require.Empty(t, mock.calls[0].arguments["password"])
	require.False(t, mock.calls[0].arguments["allowEmptyPassword"].(bool))
}

// Tests that an error is returned if the binding operation fails.
func TestBindAsUserWithError(t *testing.T) {
	// Arrange
	mock := newMockDriver()
	mock.returnSimpleBind = errTest
	settings := ldap.Settings{}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	err := controller.BindAsMaintenanceUser()

	// Assert
	require.ErrorContains(t, err, "test error")
}

// Tests that an error is returned if the searching for user group membership
// fails.
func TestSearchForUserGroupMembershipWithError(t *testing.T) {
	// Arrange
	mock := newMockDriver()
	mock.returnSearchErr = errTest
	settings := ldap.Settings{}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	groups, isAllowed, err := controller.SearchForUserGroupMembership("foo")

	// Assert
	require.Error(t, err)
	require.Nil(t, groups)
	require.False(t, isAllowed)
}

// Tests that the isAllowed status is true if the user is a member of the
// allow group.
func TestSearchForUserGroupMembershipHasAllowGroup(t *testing.T) {
	// Arrange
	mock := newMockDriver()
	mock.returnSearchValue = &goldap.SearchResult{
		Entries: []*goldap.Entry{
			{
				Attributes: []*goldap.EntryAttribute{
					{
						Name:   "groupName",
						Values: []string{"optional"},
					},
				},
			},
			{
				Attributes: []*goldap.EntryAttribute{
					{
						Name:   "groupName",
						Values: []string{"mandatory"},
					},
				},
			},
		},
	}
	settings := ldap.Settings{
		AttributeNames: ldap.LDAPAttributeNames{
			GroupCommonName: "groupName",
		},
		MandatoryAllowGroup: "mandatory",
	}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	groups, isAllowed, err := controller.SearchForUserGroupMembership("foo")

	// Assert
	require.NoError(t, err)
	require.True(t, isAllowed)
	require.Empty(t, groups)
}

// Tests that the isAllowed status is false if the user is not a member of the
// allow group.
func TestSearchForUserGroupMembershipHasNoAllowGroup(t *testing.T) {
	// Arrange
	mock := newMockDriver()
	mock.returnSearchValue = &goldap.SearchResult{
		Entries: []*goldap.Entry{
			{
				Attributes: []*goldap.EntryAttribute{
					{
						Name:   "groupName",
						Values: []string{"optional"},
					},
				},
			},
		},
	}
	settings := ldap.Settings{
		AttributeNames: ldap.LDAPAttributeNames{
			GroupCommonName: "groupName",
		},
		MandatoryAllowGroup: "mandatory",
	}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	groups, isAllowed, err := controller.SearchForUserGroupMembership("foo")

	// Assert
	require.NoError(t, err)
	require.False(t, isAllowed)
	require.Empty(t, groups)
}

// Tests that the LDAP groups are mapped to hook-specific groups.
func TestSearchForUserGroupMembershipMapping(t *testing.T) {
	// Arrange
	mock := newMockDriver()

	settings := ldap.Settings{
		AttributeNames: ldap.LDAPAttributeNames{
			GroupCommonName: "groupName",
		},
		GroupMapping: ldap.GroupMapping{
			Admin:      []string{"administrator", "company-admins,bar"},
			SuperAdmin: []string{"super-administrator", "company-super-admins"},
			ReadOnly:   []string{"read-only", "company-read-only"},
		},
	}
	controller := ldap.NewLDAPController(settings, mock)

	t.Run("none", func(t *testing.T) {
		// Act
		mock.returnSearchValue = &goldap.SearchResult{
			Entries: []*goldap.Entry{},
		}

		groups, isAllowed, err := controller.SearchForUserGroupMembership("foo")

		// Assert
		require.NoError(t, err)
		require.False(t, isAllowed)
		require.NotNil(t, groups)
		require.Empty(t, groups)
	})

	t.Run("admin", func(t *testing.T) {
		mock.returnSearchValue = &goldap.SearchResult{
			Entries: []*goldap.Entry{
				{
					Attributes: []*goldap.EntryAttribute{
						{
							Name:   "groupName",
							Values: []string{"administrator"},
						},
					},
				},
			},
		}

		// Act
		groups, isAllowed, err := controller.SearchForUserGroupMembership("foo")

		// Assert
		require.NoError(t, err)
		require.False(t, isAllowed)
		require.Len(t, groups, 1)
		require.Contains(t, groups, authdata.UserGroupIDAdmin)
	})

	t.Run("super-admin", func(t *testing.T) {
		mock.returnSearchValue = &goldap.SearchResult{
			Entries: []*goldap.Entry{
				{
					Attributes: []*goldap.EntryAttribute{
						{
							Name:   "groupName",
							Values: []string{"super-administrator"},
						},
					},
				},
			},
		}

		// Act
		groups, isAllowed, err := controller.SearchForUserGroupMembership("foo")

		// Assert
		require.NoError(t, err)
		require.False(t, isAllowed)
		require.Len(t, groups, 1)
		require.Contains(t, groups, authdata.UserGroupIDSuperAdmin)
	})

	t.Run("read-only", func(t *testing.T) {
		mock.returnSearchValue = &goldap.SearchResult{
			Entries: []*goldap.Entry{
				{
					Attributes: []*goldap.EntryAttribute{
						{
							Name:   "groupName",
							Values: []string{"read-only"},
						},
					},
				},
			},
		}

		// Act
		groups, isAllowed, err := controller.SearchForUserGroupMembership("foo")

		// Assert
		require.NoError(t, err)
		require.False(t, isAllowed)
		require.Len(t, groups, 1)
		require.Contains(t, groups, authdata.UserGroupIDReadOnly)
	})

	t.Run("admin and super-admin", func(t *testing.T) {
		mock.returnSearchValue = &goldap.SearchResult{
			Entries: []*goldap.Entry{
				{
					Attributes: []*goldap.EntryAttribute{
						{
							Name:   "groupName",
							Values: []string{"super-administrator"},
						},
					},
				},
				{
					Attributes: []*goldap.EntryAttribute{
						{
							Name:   "groupName",
							Values: []string{"administrator"},
						},
					},
				},
			},
		}

		// Act
		groups, isAllowed, err := controller.SearchForUserGroupMembership("foo")

		// Assert
		require.NoError(t, err)
		require.False(t, isAllowed)
		require.Len(t, groups, 2)
		require.Contains(t, groups, authdata.UserGroupIDSuperAdmin)
		require.Contains(t, groups, authdata.UserGroupIDAdmin)
	})

	t.Run("multiple groups - admin", func(t *testing.T) {
		mock.returnSearchValue = &goldap.SearchResult{
			Entries: []*goldap.Entry{
				{
					Attributes: []*goldap.EntryAttribute{
						{
							Name:   "groupName",
							Values: []string{"company-admins,bar"},
						},
					},
				},
			},
		}

		// Act
		groups, isAllowed, err := controller.SearchForUserGroupMembership("foo")

		// Assert
		require.NoError(t, err)
		require.False(t, isAllowed)
		require.Len(t, groups, 1)
		require.Contains(t, groups, authdata.UserGroupIDAdmin)
	})

	t.Run("multiple groups - super-admin", func(t *testing.T) {
		mock.returnSearchValue = &goldap.SearchResult{
			Entries: []*goldap.Entry{
				{
					Attributes: []*goldap.EntryAttribute{
						{
							Name:   "groupName",
							Values: []string{"company-super-admins"},
						},
					},
				},
			},
		}

		// Act
		groups, isAllowed, err := controller.SearchForUserGroupMembership("foo")

		// Assert
		require.NoError(t, err)
		require.False(t, isAllowed)
		require.Len(t, groups, 1)
		require.Contains(t, groups, authdata.UserGroupIDSuperAdmin)
	})

	t.Run("multiple groups - read-only", func(t *testing.T) {
		mock.returnSearchValue = &goldap.SearchResult{
			Entries: []*goldap.Entry{
				{
					Attributes: []*goldap.EntryAttribute{
						{
							Name:   "groupName",
							Values: []string{"company-read-only"},
						},
					},
				},
			},
		}

		// Act
		groups, isAllowed, err := controller.SearchForUserGroupMembership("foo")

		// Assert
		require.NoError(t, err)
		require.False(t, isAllowed)
		require.Len(t, groups, 1)
		require.Contains(t, groups, authdata.UserGroupIDReadOnly)
	})

	t.Run("multiple groups - admin and super-admin", func(t *testing.T) {
		mock.returnSearchValue = &goldap.SearchResult{
			Entries: []*goldap.Entry{
				{
					Attributes: []*goldap.EntryAttribute{
						{
							Name:   "groupName",
							Values: []string{"company-super-admins"},
						},
					},
				},
				{
					Attributes: []*goldap.EntryAttribute{
						{
							Name:   "groupName",
							Values: []string{"company-admins,bar"},
						},
					},
				},
			},
		}

		// Act
		groups, isAllowed, err := controller.SearchForUserGroupMembership("foo")

		// Assert
		require.NoError(t, err)
		require.False(t, isAllowed)
		require.Len(t, groups, 2)
		require.Contains(t, groups, authdata.UserGroupIDSuperAdmin)
		require.Contains(t, groups, authdata.UserGroupIDAdmin)
	})
}
