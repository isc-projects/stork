package agent

import (
	"path"
	reflect "reflect"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"isc.org/stork/hooks"
	"isc.org/stork/hooks/agent/forwardtokeaoverhttpcallouts"
	"isc.org/stork/testutil"
)

// Carrier mock interface for mockgen.
type beforeForwardToKeaOverHTTPCalloutCarrier interface { //nolint:unused
	forwardtokeaoverhttpcallouts.BeforeForwardToKeaOverHTTPCallouts
	hooks.CalloutCarrier
}

//go:generate mockgen -source hook_test.go -package=agent -destination=hookmock_test.go -mock_names=beforeForwardToKeaOverHTTPCalloutCarrier=MockBeforeForwardToKeaOverHTTPCalloutCarrier isc.org/agent beforeForwardToKeaOverHTTPCalloutCarrier

// Test that the hook manager is constructed properly.
func TestNewHookManager(t *testing.T) {
	// Arrange & Act
	hookManager := NewHookManager()

	// Assert
	require.NotNil(t, hookManager)
	supportedTypes := hookManager.HookManager.GetExecutor().GetTypesOfSupportedCalloutSpecifications()
	require.Len(t, supportedTypes, 1)
}

// Test that constructing the hook manager from the directory fails if the
// directory doesn't exist.
func TestHookManagerFromDirectoryReturnErrorOnInvalidDirectory(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()
	hookManager := NewHookManager()

	// Arrange & Act
	err := hookManager.RegisterHooksFromDirectory(
		"foo",
		path.Join(sb.BasePath, "not-exists-directory"),
		map[string]hooks.HookSettings{},
	)

	// Assert
	require.Error(t, err)
}

// Test that the hook manager is constructed properly from the callout carrier
// slice.
func TestHookManagerFromCallouts(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockBeforeForwardToKeaOverHTTPCalloutCarrier(ctrl)

	hookManager := NewHookManager()

	// Act
	hookManager.RegisterCalloutCarriers([]hooks.CalloutCarrier{
		mock,
	})

	// Assert
	require.True(t, hookManager.GetExecutor().HasRegistered(reflect.TypeOf((*forwardtokeaoverhttpcallouts.BeforeForwardToKeaOverHTTPCallouts)(nil)).Elem()))
}

// Test that the hook manager is closing properly.
func TestHookManagerClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockBeforeForwardToKeaOverHTTPCalloutCarrier(ctrl)
	mock.
		EXPECT().
		Close().
		Return(nil).
		Times(1)

	hookManager := NewHookManager()
	hookManager.RegisterCalloutCarriers([]hooks.CalloutCarrier{
		mock,
	})

	// Act
	err := hookManager.Close()

	// Assert
	require.NoError(t, err)
}

// Test that the hook manager passes through the error from the callout carrier
// and the error raising doesn't interrupt the close operation.
func TestHookManagerCloseWithErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWithoutErr1 := NewMockBeforeForwardToKeaOverHTTPCalloutCarrier(ctrl)
	mockWithoutErr1.
		EXPECT().
		Close().
		Return(nil).
		Times(1)

	mockWithoutErr2 := NewMockBeforeForwardToKeaOverHTTPCalloutCarrier(ctrl)
	mockWithoutErr2.
		EXPECT().
		Close().
		Return(nil).
		Times(1)

	mockWithErr := NewMockBeforeForwardToKeaOverHTTPCalloutCarrier(ctrl)
	mockWithErr.
		EXPECT().
		Close().
		Return(errors.New("foo")).
		Times(1)

	hookManager := NewHookManager()
	hookManager.RegisterCalloutCarriers([]hooks.CalloutCarrier{
		mockWithoutErr1,
		mockWithErr,
		mockWithoutErr2,
	})

	// Act
	err := hookManager.Close()

	// Assert
	require.ErrorContains(t, err, "foo")
}
