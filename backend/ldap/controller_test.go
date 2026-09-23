package ldap_test

import (
	"crypto/tls"
	"errors"
	"testing"
	"time"

	goldap "github.com/go-ldap/ldap/v3"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"isc.org/stork/ldap"
	"isc.org/stork/server/authdata"
)

// Base error for the testing purposes.
var errTest = errors.New("test error")

//go:generate mockgen -package=ldap_test -destination=drivermock_test.go isc.org/stork/ldap LDAPDriver

// Tests if the LDAP controller can be created.
func TestNewLDAPController(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Arrange & Act
	controller := ldap.NewLDAPController(ldap.Settings{}, NewMockLDAPDriver(ctrl))

	// Assert
	require.NotNil(t, controller)
}

// Tests if the LDAP controller is not configured with TLS if LDAP server is
// served over unsecure protocol.
func TestConnectWithoutTLS(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	var dialURL string
	var dialTLS *tls.Config
	var dialTimeout time.Duration
	mock.EXPECT().Dial(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(url string, tlsConfig *tls.Config, timeout time.Duration) error {
		dialURL = url
		dialTLS = tlsConfig
		dialTimeout = timeout
		return nil
	})
	settings := ldap.Settings{
		DialURL: "ldap://foobar:42",
		Timeout: 24 * time.Hour,
	}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	err := controller.Connect()

	// Assert
	require.NoError(t, err)
	require.Equal(t, "ldap://foobar:42", dialURL)
	require.Equal(t, 24*time.Hour, dialTimeout)
	require.Nil(t, dialTLS)
}

// Tests that the LDAP controller returns an error if the connection cannot be
// established.
func TestConnectWithError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	mock.EXPECT().Dial("ldap://foobar:42", gomock.Nil(), 24*time.Hour).Return(errTest)
	settings := ldap.Settings{
		DialURL: "ldap://foobar:42",
		Timeout: 24 * time.Hour,
	}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	err := controller.Connect()

	// Assert
	require.Error(t, err)
}

// Tests that the LDAP controller can be configured to skip TLS server
// verification.
func TestConnectWithTLSSkipVerification(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	var dialTLS *tls.Config
	mock.EXPECT().Dial("ldaps://foobar:42", gomock.Any(), 24*time.Hour).DoAndReturn(func(_ string, tlsConfig *tls.Config, _ time.Duration) error {
		dialTLS = tlsConfig
		return nil
	})
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
	require.NotNil(t, dialTLS)
	require.True(t, dialTLS.InsecureSkipVerify)
}

// Tests that the LDAP controller is configured with TLS if LDAP server is
// served over secure protocol.
func TestConnectWithTLS(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	var dialTLS *tls.Config
	mock.EXPECT().Dial("ldaps://foobar:42", gomock.Any(), 24*time.Hour).DoAndReturn(func(_ string, tlsConfig *tls.Config, _ time.Duration) error {
		dialTLS = tlsConfig
		return nil
	})
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
	require.NotNil(t, dialTLS)
	require.False(t, dialTLS.InsecureSkipVerify)
}

// Tests that the LDAP controller binds the maintenance user with the proper
// credentials.
func TestBindAsMaintenanceUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	mock.EXPECT().SimpleBind("cn=bar", "boz", false).Return(nil)
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
}

// Tests that the maintenance user can be bound with an empty password.
func TestBindAsMaintenanceUserWithEmptyPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	mock.EXPECT().SimpleBind("cn=bar", "", true).Return(nil)
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
}

// Tests that the binding of the maintenance user fails if the connection to
// the LDAP server cannot be established.
func TestBindAsMaintenanceUserWithError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	mock.EXPECT().SimpleBind("", "", true).Return(errTest)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	searchResult := &goldap.SearchResult{
		Entries: []*goldap.Entry{
			{DN: "foo"},
		},
	}
	var request *goldap.SearchRequest
	mock.EXPECT().Search(gomock.Any()).DoAndReturn(func(r *goldap.SearchRequest) (*goldap.SearchResult, error) {
		request = r
		return searchResult, nil
	})
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	mock.EXPECT().Search(gomock.Any()).Return(&goldap.SearchResult{
		Entries: []*goldap.Entry{},
	}, nil)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	mock.EXPECT().Search(gomock.Any()).Return(&goldap.SearchResult{
		Entries: []*goldap.Entry{
			{DN: "foo"},
			{DN: "boz"},
		},
	}, nil)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	mock.EXPECT().Search(gomock.Any()).Return(&goldap.SearchResult{
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
	}, nil)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	mock.EXPECT().Search(gomock.Any()).Return(&goldap.SearchResult{
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
	}, nil)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	mock.EXPECT().Search(gomock.Any()).Return(&goldap.SearchResult{
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
	}, nil)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	mock.EXPECT().Search(gomock.Any()).Return(&goldap.SearchResult{
		Entries: []*goldap.Entry{
			{
				DN:         "cn=foobar,dc=example,dc=com",
				Attributes: []*goldap.EntryAttribute{},
			},
		},
	}, nil)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	mock.EXPECT().Search(gomock.Any()).Return(&goldap.SearchResult{
		Entries: []*goldap.Entry{},
	}, nil)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	mock.EXPECT().Search(gomock.Any()).Return(nil, errTest)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	mock.EXPECT().SimpleBind("foo", "bar", false).Return(nil)
	settings := ldap.Settings{}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	err := controller.BindAsUser("foo", "bar")

	// Assert
	require.NoError(t, err)
}

// Tests that the LDAP controller can bind as an arbitrary user with an empty
// password.
func TestBindAsUserWithEmptyPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	mock.EXPECT().SimpleBind("foo", "", false).Return(nil)
	settings := ldap.Settings{}
	controller := ldap.NewLDAPController(settings, mock)

	// Act
	err := controller.BindAsUser("foo", "")

	// Assert
	// There is an error in runtime because empty password is not allowed.
	require.NoError(t, err)
}

// Tests that an error is returned if the binding operation fails.
func TestBindAsUserWithError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	mock.EXPECT().SimpleBind("", "", true).Return(errTest)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	mock.EXPECT().Search(gomock.Any()).Return(nil, errTest)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	mock.EXPECT().Search(gomock.Any()).Return(&goldap.SearchResult{
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
	}, nil)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	mock.EXPECT().Search(gomock.Any()).Return(&goldap.SearchResult{
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
	}, nil)
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

	runCase := func(name string, entries []*goldap.Entry, expectedGroups []authdata.UserGroupID) {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mock := NewMockLDAPDriver(ctrl)
			mock.EXPECT().Search(gomock.Any()).Return(&goldap.SearchResult{Entries: entries}, nil)
			controller := ldap.NewLDAPController(settings, mock)

			// Act
			groups, isAllowed, err := controller.SearchForUserGroupMembership("foo")

			// Assert
			require.NoError(t, err)
			require.False(t, isAllowed)
			require.Len(t, groups, len(expectedGroups))
			for _, group := range expectedGroups {
				require.Contains(t, groups, group)
			}
		})
	}

	runCase("none", []*goldap.Entry{}, nil)

	runCase("admin", []*goldap.Entry{
		{
			Attributes: []*goldap.EntryAttribute{{
				Name:   "groupName",
				Values: []string{"administrator"},
			}},
		},
	}, []authdata.UserGroupID{authdata.UserGroupIDAdmin})

	runCase("super-admin", []*goldap.Entry{
		{
			Attributes: []*goldap.EntryAttribute{{
				Name:   "groupName",
				Values: []string{"super-administrator"},
			}},
		},
	}, []authdata.UserGroupID{authdata.UserGroupIDSuperAdmin})

	runCase("read-only", []*goldap.Entry{
		{
			Attributes: []*goldap.EntryAttribute{{
				Name:   "groupName",
				Values: []string{"read-only"},
			}},
		},
	}, []authdata.UserGroupID{authdata.UserGroupIDReadOnly})

	runCase("admin and super-admin", []*goldap.Entry{
		{
			Attributes: []*goldap.EntryAttribute{{
				Name:   "groupName",
				Values: []string{"super-administrator"},
			}},
		},
		{
			Attributes: []*goldap.EntryAttribute{{
				Name:   "groupName",
				Values: []string{"administrator"},
			}},
		},
	}, []authdata.UserGroupID{authdata.UserGroupIDSuperAdmin, authdata.UserGroupIDAdmin})

	runCase("multiple groups - admin", []*goldap.Entry{
		{
			Attributes: []*goldap.EntryAttribute{{
				Name:   "groupName",
				Values: []string{"company-admins,bar"},
			}},
		},
	}, []authdata.UserGroupID{authdata.UserGroupIDAdmin})

	runCase("multiple groups - super-admin", []*goldap.Entry{
		{
			Attributes: []*goldap.EntryAttribute{{
				Name:   "groupName",
				Values: []string{"company-super-admins"},
			}},
		},
	}, []authdata.UserGroupID{authdata.UserGroupIDSuperAdmin})

	runCase("multiple groups - read-only", []*goldap.Entry{
		{
			Attributes: []*goldap.EntryAttribute{{
				Name:   "groupName",
				Values: []string{"company-read-only"},
			}},
		},
	}, []authdata.UserGroupID{authdata.UserGroupIDReadOnly})

	runCase("multiple groups - admin and super-admin", []*goldap.Entry{
		{
			Attributes: []*goldap.EntryAttribute{{
				Name:   "groupName",
				Values: []string{"company-super-admins"},
			}},
		},
		{
			Attributes: []*goldap.EntryAttribute{{
				Name:   "groupName",
				Values: []string{"company-admins,bar"},
			}},
		},
	}, []authdata.UserGroupID{authdata.UserGroupIDSuperAdmin, authdata.UserGroupIDAdmin})
}
