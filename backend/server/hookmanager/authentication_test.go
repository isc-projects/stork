package hookmanager

import (
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"isc.org/stork/hooks"
	"isc.org/stork/hooks/server/authenticationcallout"
)

//go:generate mockgen -package=hookmanager -destination=hookmanager_mock.go isc.org/stork/hooks/server/authenticationcallout AuthenticationCallout

// Test that the authentication hook is detected properly.
func TestHasAuthenticationHook(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockAuthenticationCallout(ctrl)

	hookManager := NewHookManager()
	hookManager.RegisterCallouts([]hooks.Callout{mock})

	// Act
	hasHook := hookManager.HasAuthenticationHook()

	// Assert
	require.True(t, hasHook)
}

// Test that the authentication callout is called.
func TestAuthenticate(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	username := "foo"
	password := "bar"

	mock := NewMockAuthenticationCallout(ctrl)
	mock.EXPECT().
		Authenticate(gomock.Any(), gomock.Any(), &username, &password).
		Return(&authenticationcallout.User{
			ID:       42,
			Login:    "foo",
			Email:    "foo@example.com",
			Lastname: "oof",
			Name:     "ofo",
			Groups:   []int{1, 2, 3},
		}, nil).
		Times(1)

	hookManager := NewHookManager()
	hookManager.RegisterCallouts([]hooks.Callout{mock})

	// Act
	user, err := hookManager.Authenticate(nil, nil, &username, &password)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "foo@example.com", user.Email)
}

// Test that only first authentication callout is called.
func TestAuthenticateIsSingle(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock1 := NewMockAuthenticationCallout(ctrl)
	mock1.EXPECT().
		Authenticate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&authenticationcallout.User{}, nil).
		Times(1)

	mock2 := NewMockAuthenticationCallout(ctrl)
	mock2.EXPECT().
		Authenticate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&authenticationcallout.User{}, nil).
		Times(0)

	hookManager := NewHookManager()
	hookManager.RegisterCallouts([]hooks.Callout{mock1, mock2})

	// Act
	user, err := hookManager.Authenticate(nil, nil, nil, nil)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, user)
}

// Test that the authentication callout returns a default value if no callouts
// are registered.
func TestAuthenticateDefault(t *testing.T) {
	// Arrange
	hookManager := NewHookManager()

	// Act
	user, err := hookManager.Authenticate(nil, nil, nil, nil)

	// Assert
	require.NoError(t, err)
	require.Nil(t, user)
}
