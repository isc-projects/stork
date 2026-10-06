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
	mock.EXPECT().Close()
	settings := ldap.Settings{
		DialURL: "ldap://foobar:42",
		Timeout: 24 * time.Hour,
	}
	controller := ldap.NewLDAPController(settings, mock)
	defer controller.Close()

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
	mock.EXPECT().Close()
	settings := ldap.Settings{
		DialURL: "ldap://foobar:42",
		Timeout: 24 * time.Hour,
	}
	controller := ldap.NewLDAPController(settings, mock)
	defer controller.Close()

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
	mock.EXPECT().Close()
	settings := ldap.Settings{
		DialURL:                   "ldaps://foobar:42",
		Timeout:                   24 * time.Hour,
		TLSSkipServerVerification: true,
	}
	controller := ldap.NewLDAPController(settings, mock)
	defer controller.Close()

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
	mock.EXPECT().Close()
	settings := ldap.Settings{
		DialURL:                   "ldaps://foobar:42",
		Timeout:                   24 * time.Hour,
		TLSSkipServerVerification: false,
	}
	controller := ldap.NewLDAPController(settings, mock)
	defer controller.Close()

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
	mock.EXPECT().Close()
	settings := ldap.Settings{
		Root:         "foo",
		BindUserDN:   "cn=bar",
		BindPassword: "boz",
	}
	controller := ldap.NewLDAPController(settings, mock)
	defer controller.Close()

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
	mock.EXPECT().Close()
	settings := ldap.Settings{
		Root:         "foo",
		BindUserDN:   "cn=bar",
		BindPassword: "",
	}
	controller := ldap.NewLDAPController(settings, mock)
	defer controller.Close()

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
	mock.EXPECT().Close()
	settings := ldap.Settings{}
	controller := ldap.NewLDAPController(settings, mock)
	defer controller.Close()

	// Act
	err := controller.BindAsMaintenanceUser()

	// Assert
	require.Error(t, err)
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
	mock.EXPECT().Close()
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
	defer controller.Close()

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
	mock.EXPECT().Close()
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
	defer controller.Close()

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
	mock.EXPECT().Close()
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
	defer controller.Close()

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
	mock.EXPECT().Close()
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
	defer controller.Close()

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
	mock.EXPECT().Close()
	settings := ldap.Settings{}
	controller := ldap.NewLDAPController(settings, mock)
	defer controller.Close()

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
	mock.EXPECT().Close()
	settings := ldap.Settings{}
	controller := ldap.NewLDAPController(settings, mock)
	defer controller.Close()

	// Act
	userProfile, err := controller.SearchForUserProfile("username")

	// Assert
	require.ErrorContains(t, err, "test error")
	require.Nil(t, userProfile)
}

// Tests that an error is returned if the binding operation fails.
func TestBindAsUserWithError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockLDAPDriver(ctrl)
	mock.EXPECT().SimpleBind("", "", true).Return(errTest)
	mock.EXPECT().Close()
	settings := ldap.Settings{}
	controller := ldap.NewLDAPController(settings, mock)
	defer controller.Close()

	// Act
	err := controller.BindAsMaintenanceUser()

	// Assert
	require.ErrorContains(t, err, "test error")
}
