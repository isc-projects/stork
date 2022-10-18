package hookmanager

import (
	"context"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"isc.org/stork/hooks"
)

//go:generate mockgen -package=hookmanager -destination=hookmanager_mock.go isc.org/stork/server/callouts/authenticationcallout AuthenticationCallout,User

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

	mockUser := NewMockUser(ctrl)
	mockUserRecorder := mockUser.EXPECT()
	mockUserRecorder.GetEmail().Return("foo@example.com")
	mockUserRecorder.GetGroups().Return([]int{1, 2, 3})
	mockUserRecorder.GetID().Return(42)
	mockUserRecorder.GetLastName().Return("oof")
	mockUserRecorder.GetLogin().Return("foo")
	mockUserRecorder.GetName().Return("ofo")

	mockCallout := NewMockAuthenticationCallout(ctrl)
	mockCallout.EXPECT().
		Authenticate(gomock.Any(), gomock.Any(), &username, &password).
		Return(mockUser, nil).
		Times(1)

	hookManager := NewHookManager()
	hookManager.RegisterCallouts([]hooks.Callout{mockCallout})

	// Act
	user, err := hookManager.Authenticate(context.Background(), nil, &username, &password)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "foo@example.com", user.GetEmail())
}

// Test that only first authentication callout is called.
func TestAuthenticateIsSingle(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock1 := NewMockAuthenticationCallout(ctrl)
	mock1.EXPECT().
		Authenticate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(NewMockUser(ctrl), nil).
		Times(1)

	mock2 := NewMockAuthenticationCallout(ctrl)
	mock2.EXPECT().
		Authenticate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(NewMockUser(ctrl), nil).
		Times(0)

	hookManager := NewHookManager()
	hookManager.RegisterCallouts([]hooks.Callout{mock1, mock2})

	// Act
	user, err := hookManager.Authenticate(context.Background(), nil, nil, nil)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, user)
}

// Test that the error is returned if the authentication fails.
func TestAuthenticateReturnError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock1 := NewMockAuthenticationCallout(ctrl)
	mock1.EXPECT().
		Authenticate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("foo")).
		Times(1)

	mock2 := NewMockAuthenticationCallout(ctrl)
	mock2.EXPECT().
		Authenticate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("bar")).
		Times(0)

	hookManager := NewHookManager()
	hookManager.RegisterCallouts([]hooks.Callout{mock1, mock2})

	// Act
	user, err := hookManager.Authenticate(context.Background(), nil, nil, nil)

	// Assert
	require.ErrorContains(t, err, "foo")
	require.Nil(t, user)
}

// Test that the authentication callout returns a default value if no callouts
// are registered.
func TestAuthenticateDefault(t *testing.T) {
	// Arrange
	hookManager := NewHookManager()

	// Act
	user, err := hookManager.Authenticate(context.Background(), nil, nil, nil)

	// Assert
	require.NoError(t, err)
	require.Nil(t, user)
}

// Test that the unauthenticate function is called only once.
func TestUnauthenticate(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock1 := NewMockAuthenticationCallout(ctrl)
	mock1.EXPECT().
		Unauthenticate(gomock.Any()).
		Return(nil).
		Times(1)

	mock2 := NewMockAuthenticationCallout(ctrl)
	mock2.EXPECT().
		Unauthenticate(gomock.Any()).
		Return(nil).
		Times(0)

	hookManager := NewHookManager()
	hookManager.RegisterCallouts([]hooks.Callout{mock1, mock2})

	// Act
	err := hookManager.Unauthenticate(context.Background())

	// Assert
	require.NoError(t, err)
}

// Test that the unauthenticate function returns an error properly.
func TestUnauthenticateError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock1 := NewMockAuthenticationCallout(ctrl)
	mock1.EXPECT().
		Unauthenticate(gomock.Any()).
		Return(errors.New("foo")).
		Times(1)

	mock2 := NewMockAuthenticationCallout(ctrl)
	mock2.EXPECT().
		Unauthenticate(gomock.Any()).
		Return(errors.New("bar")).
		Times(0)

	hookManager := NewHookManager()
	hookManager.RegisterCallouts([]hooks.Callout{mock1, mock2})

	// Act
	err := hookManager.Unauthenticate(context.Background())

	// Assert
	require.ErrorContains(t, err, "foo")
}
