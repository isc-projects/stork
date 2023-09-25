package hookmanager

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"isc.org/stork/hooks"
	"isc.org/stork/hooks/server/authenticationcallouts"
)

// Carrier mock interface for mockgen.
type authenticationCalloutCarrier interface { //nolint:unused
	authenticationcallouts.AuthenticationCallouts
	hooks.CalloutCarrier
}

//go:generate mockgen -package=hookmanager -destination=authenticationcalloutcarriermock_test.go -source=authentication_test.go -mock_names=authenticationCalloutCarrier=MockAuthenticationCalloutCarrier isc.org/server/hookmanager authenticationCalloutCarrier
//go:generate mockgen -package=hookmanager -destination=authenticationcalloutsmock_test.go -source=../../hooks/server/authenticationcallouts/authenticationcallouts.go isc.org/server/hookmanager AuthenticationMetadata

// Test that the authentication callout is called.
func TestAuthenticate(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	identifier := "foo"
	secret := "bar"

	metadataMock := NewMockAuthenticationMetadata(ctrl)
	metadataMock.EXPECT().
		GetID().
		Return("mock")

	mock := NewMockAuthenticationCalloutCarrier(ctrl)
	mock.EXPECT().
		Authenticate(gomock.Any(), gomock.Any(), &identifier, &secret).
		Return(&authenticationcallouts.User{
			ID:       "42",
			Login:    "foo",
			Email:    "foo@example.com",
			Lastname: "oof",
			Name:     "ofo",
			Groups:   []authenticationcallouts.UserGroupID{1, 2, 3},
		}, nil).
		Times(1)
	mock.EXPECT().
		GetMetadata().
		Return(metadataMock)

	hookManager := NewHookManager()
	hookManager.RegisterCalloutCarrier(mock)

	// Act
	user, err := hookManager.Authenticate(context.Background(), nil, "mock", &identifier, &secret)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "foo@example.com", user.Email)
}

// Test that only first matching authentication callout is called.
func TestAuthenticateOnlyFirst(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	metadataMock := NewMockAuthenticationMetadata(ctrl)
	metadataMock.EXPECT().
		GetID().
		Return("mock")

	mock1 := NewMockAuthenticationCalloutCarrier(ctrl)
	mock1.EXPECT().
		Authenticate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&authenticationcallouts.User{}, nil).
		Times(1)
	mock1.EXPECT().
		GetMetadata().
		Return(metadataMock)

	mock2 := NewMockAuthenticationCalloutCarrier(ctrl)
	mock2.EXPECT().
		Authenticate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&authenticationcallouts.User{}, nil).
		Times(0)

	hookManager := NewHookManager()
	hookManager.RegisterCalloutCarriers([]hooks.CalloutCarrier{mock1, mock2})

	// Act
	user, err := hookManager.Authenticate(context.Background(), nil, "mock", nil, nil)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, user)
}

// Test that the error is returned if the authentication fails.
func TestAuthenticateReturnError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	metadataMock1 := NewMockAuthenticationMetadata(ctrl)
	metadataMock1.EXPECT().
		GetID().
		Return("mock1")

	mock1 := NewMockAuthenticationCalloutCarrier(ctrl)
	mock1.EXPECT().
		Authenticate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("foo")).
		Times(1)
	mock1.EXPECT().
		GetMetadata().
		Return(metadataMock1)

	mock2 := NewMockAuthenticationCalloutCarrier(ctrl)
	mock2.EXPECT().
		Authenticate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("bar")).
		Times(0)

	hookManager := NewHookManager()
	hookManager.RegisterCalloutCarriers([]hooks.CalloutCarrier{mock1, mock2})

	// Act
	user, err := hookManager.Authenticate(context.Background(), nil, "mock1", nil, nil)

	// Assert
	require.ErrorContains(t, err, "foo")
	require.Nil(t, user)
}

// Test that the authentication callout returns a default value (nil) if no callouts
// are registered.
func TestAuthenticateDefault(t *testing.T) {
	// Arrange
	hookManager := NewHookManager()

	// Act
	user, err := hookManager.Authenticate(context.Background(), nil, "internal", nil, nil)

	// Assert
	require.ErrorContains(t, err, "authentication method is not supported")
	require.Nil(t, user)
}

// Test that the unauthenticate function is called only once.
func TestUnauthenticate(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	metadataMock1 := NewMockAuthenticationMetadata(ctrl)
	metadataMock1.EXPECT().
		GetID().
		Return("mock1")

	mock1 := NewMockAuthenticationCalloutCarrier(ctrl)
	mock1.EXPECT().
		Unauthenticate(gomock.Any()).
		Return(nil).
		Times(1)
	mock1.EXPECT().
		GetMetadata().
		Return(metadataMock1)

	mock2 := NewMockAuthenticationCalloutCarrier(ctrl)
	mock2.EXPECT().
		Unauthenticate(gomock.Any()).
		Return(nil).
		Times(0)

	hookManager := NewHookManager()
	hookManager.RegisterCalloutCarriers([]hooks.CalloutCarrier{mock1, mock2})

	// Act
	err := hookManager.Unauthenticate(context.Background(), "mock1")

	// Assert
	require.NoError(t, err)
}

// Test that the unauthenticate function returns an error properly.
func TestUnauthenticateError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	metadataMock1 := NewMockAuthenticationMetadata(ctrl)
	metadataMock1.EXPECT().
		GetID().
		Return("mock1")

	mock1 := NewMockAuthenticationCalloutCarrier(ctrl)
	mock1.EXPECT().
		Unauthenticate(gomock.Any()).
		Return(errors.New("foo")).
		Times(1)
	mock1.EXPECT().
		GetMetadata().
		Return(metadataMock1)

	mock2 := NewMockAuthenticationCalloutCarrier(ctrl)
	mock2.EXPECT().
		Unauthenticate(gomock.Any()).
		Return(errors.New("bar")).
		Times(0)

	hookManager := NewHookManager()
	hookManager.RegisterCalloutCarriers([]hooks.CalloutCarrier{mock1, mock2})

	// Act
	err := hookManager.Unauthenticate(context.Background(), "mock1")

	// Assert
	require.ErrorContains(t, err, "foo")
}

// Test that all authentication metadata are returned.
func TestGetAuthenticationMetadata(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var mocks []hooks.CalloutCarrier

	for i := 0; i < 3; i++ {
		metadataMock := NewMockAuthenticationMetadata(ctrl)
		metadataMock.EXPECT().
			GetID().
			Return(fmt.Sprintf("mock-%d", i))

		mock := NewMockAuthenticationCalloutCarrier(ctrl)
		mock.EXPECT().
			GetMetadata().
			Return(metadataMock).
			Times(1)

		mocks = append(mocks, mock)
	}

	hookManager := NewHookManager()
	hookManager.RegisterCalloutCarriers(mocks)

	// Act
	results := hookManager.GetAuthenticationMetadata()

	// Assert
	require.Len(t, results, 3)
	for i, result := range results {
		require.EqualValues(t, fmt.Sprintf("mock-%d", i), result.GetID())
	}
}

// Test that an empty slice is returned if no authentication carrier is
// registered.
func TestGetAuthenticationMetadataNoCalloutCarriers(t *testing.T) {
	// Arrange
	hookManager := NewHookManager()

	// Act
	results := hookManager.GetAuthenticationMetadata()

	// Assert
	require.Len(t, results, 0)
}
