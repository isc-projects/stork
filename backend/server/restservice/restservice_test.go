package restservice

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"isc.org/stork/hooks"
	authenticationcallouts "isc.org/stork/hooks/server/authenticationcallouts"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	apps "isc.org/stork/server/apps"
	appstest "isc.org/stork/server/apps/test"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/hookmanager"
	storktest "isc.org/stork/server/test"
	storktestdbmodel "isc.org/stork/server/test/dbmodel"
	"isc.org/stork/testutil"
)

// Carrier mock interface for mockgen.
type authenticationCalloutCarrier interface { //nolint:unused
	authenticationcallouts.AuthenticationCallouts
	hooks.CalloutCarrier
}

//go:generate mockgen -package=restservice -destination=authenticationcalloutsmock_test.go -source=../../hooks/server/authenticationcallouts/authenticationcallouts.go isc.org/server/hookmanager AuthenticationMetadata
//go:generate mockgen -package=restservice -destination=authenticationcalloutcarriermock_test.go -source=restservice_test.go -mock_names=authenticationCalloutCarrier=MockAuthenticationCalloutCarrier isc.org/server/hookmanager authenticationCalloutCarrier

// Tests instantiating RestAPI.
func TestNewRestAPI(t *testing.T) {
	db, dbs, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Initialize all structures that can be passed to the function
	// under test.
	settings := &RestAPISettings{}
	agents := agentcommtest.NewFakeAgents(nil, nil)
	eventcenter := &storktestdbmodel.FakeEventCenter{}
	dispatcher := &storktestdbmodel.FakeDispatcher{}
	pullers := &apps.Pullers{}
	collector := storktest.NewFakeMetricsCollector()
	configManager := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: agents,
	})
	endpointControl := NewEndpointControl()

	// Specify all supported structures.
	api, err := NewRestAPI(settings, dbs, db, agents, eventcenter, pullers, dispatcher, collector, configManager, endpointControl)
	require.NoError(t, err)
	require.NotNil(t, api)
	require.Equal(t, api.Settings, settings)
	require.Equal(t, api.DBSettings, dbs)
	require.Equal(t, api.DB, db)
	require.Equal(t, api.EventCenter, eventcenter)
	require.Equal(t, api.Pullers, pullers)
	require.Equal(t, api.ReviewDispatcher, dispatcher)
	require.Equal(t, api.MetricsCollector, collector)
	require.Equal(t, api.ConfigManager, configManager)
	require.Equal(t, api.EndpointControl, endpointControl)

	// Reverse their order.
	api, err = NewRestAPI(endpointControl, configManager, collector, dispatcher, pullers, eventcenter, agents, db, dbs, settings)
	require.NoError(t, err)
	require.NotNil(t, api)
	require.Equal(t, api.Settings, settings)
	require.Equal(t, api.DBSettings, dbs)
	require.Equal(t, api.DB, db)
	require.Equal(t, api.EventCenter, eventcenter)
	require.Equal(t, api.Pullers, pullers)
	require.Equal(t, api.ReviewDispatcher, dispatcher)
	require.Equal(t, api.MetricsCollector, collector)
	require.Equal(t, api.ConfigManager, configManager)
	require.Equal(t, api.EndpointControl, endpointControl)

	// Specify one structure and one interface.
	api, err = NewRestAPI(dbs, dispatcher)
	require.NoError(t, err)
	require.NotNil(t, api)
	require.Nil(t, api.Settings)
	require.Equal(t, api.DBSettings, dbs)
	require.Nil(t, api.DB, db)
	require.Nil(t, api.EventCenter)
	require.Nil(t, api.Pullers)
	require.Equal(t, api.ReviewDispatcher, dispatcher)
	require.Nil(t, api.MetricsCollector)

	// Pass null pointer. It should be ignored.
	pullers = nil
	api, err = NewRestAPI(dbs, nil, dispatcher, pullers)
	require.NoError(t, err)
	require.NotNil(t, api)
	require.Equal(t, api.DBSettings, dbs)
	require.Nil(t, api.Pullers)
	require.Equal(t, api.ReviewDispatcher, dispatcher)

	// Database settings are required. An error should be returned.
	api, err = NewRestAPI(dispatcher)
	require.Error(t, err)
	require.Nil(t, api)

	// Non-pointers should be rejected.
	api, err = NewRestAPI(*dbs)
	require.Error(t, err)
	require.Nil(t, api)

	// All arguments must be structures.
	num := 5
	api, err = NewRestAPI(dbs, &num)
	require.Error(t, err)
	require.Nil(t, api)

	// Unsupported structure should be rejected.
	testStruct := struct {
		num int
	}{
		num: 5,
	}
	api, err = NewRestAPI(dbs, &testStruct)
	require.Error(t, err)
	require.Nil(t, api)

	// No arguments should cause an error because settings aren't
	// specified.
	api, err = NewRestAPI()
	require.Error(t, err)
	require.Nil(t, api)
}

// Test that the authentication icons are extracted from callout carriers.
func TestPrepareAuthenticationIconsExtractFromCarriers(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()
	// Create directory structure.
	_, _ = sb.Join("assets/authentication-methods/default.png")

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	carrierMocks := []hooks.CalloutCarrier{}

	for i := 0; i < 2; i++ {
		metadataMock := NewMockAuthenticationMetadata(ctrl)
		metadataMock.EXPECT().GetID().Return(fmt.Sprintf("mock-%d", i))
		metadataMock.EXPECT().
			GetIcon().
			Return(
				io.NopCloser(
					bytes.NewReader(
						[]byte(fmt.Sprintf("mock-%d", i)),
					),
				), nil,
			)

		carrierMock := NewMockAuthenticationCalloutCarrier(ctrl)
		carrierMock.EXPECT().GetMetadata().Return(metadataMock)
		carrierMocks = append(carrierMocks, carrierMock)
	}

	hookManager := hookmanager.NewHookManager()
	hookManager.RegisterCalloutCarriers(carrierMocks)

	// Act
	err := prepareAuthenticationIcons(hookManager, sb.BasePath)

	// Assert
	require.NoError(t, err)

	for i := 0; i < 2; i++ {
		iconPath := path.Join(sb.BasePath, "assets", "authentication-methods", fmt.Sprintf("mock-%d.png", i))
		require.FileExists(t, iconPath)
		content, _ := os.ReadFile(iconPath)
		require.EqualValues(t, fmt.Sprintf("mock-%d", i), string(content))
	}
}

// Test that the error is returned if the icon directory is not writable.
func TestPrepareAuthenticationIconsNonWritableDirectory(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	sb := testutil.NewSandbox()
	defer sb.Close()

	metadataMock := NewMockAuthenticationMetadata(ctrl)
	metadataMock.EXPECT().GetID().Return("mock")
	metadataMock.EXPECT().
		GetIcon().
		Return(
			io.NopCloser(
				bytes.NewReader(
					[]byte("mock"),
				),
			), nil,
		)

	carrierMock := NewMockAuthenticationCalloutCarrier(ctrl)
	carrierMock.EXPECT().GetMetadata().Return(metadataMock)
	carrierMocks := []hooks.CalloutCarrier{carrierMock}

	hookManager := hookmanager.NewHookManager()
	hookManager.RegisterCalloutCarriers(carrierMocks)

	// Act
	err := prepareAuthenticationIcons(
		hookManager,
		path.Join(sb.BasePath, "non-existing-directory"),
	)

	// Assert
	require.ErrorContains(t, err, "cannot open the icon file to write")
}

// Test that the empty base URL leaves the existing value.
func TestSetBaseURLEmpty(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()
	filepath, _ := sb.Write("index.html", `<base href="/foo/">`)
	directory := path.Dir(filepath)

	// Act
	err := setBaseURLInIndexFile("", directory)

	// Assert
	require.NoError(t, err)
	content, _ := os.ReadFile(filepath)
	require.EqualValues(t, `<base href="/foo/">`, string(content))
}

// Test that the base HTML tag with close slash is supported.
func TestSetBaseURLTagWithCloseSlash(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()
	filepath, _ := sb.Write("index.html", `<base href="/foo/"/>`)
	directory := path.Dir(filepath)

	// Act
	err := setBaseURLInIndexFile("/bar/", directory)

	// Assert
	require.NoError(t, err)
	content, _ := os.ReadFile(filepath)
	require.EqualValues(t, `<base href="/bar/"/>`, string(content))
}

// Test that the base HTML tag without close slash is supported.
func TestSetBaseURLTagWithoutCloseSlash(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()
	filepath, _ := sb.Write("index.html", `<base href="/foo/">`)
	directory := path.Dir(filepath)

	// Act
	err := setBaseURLInIndexFile("/bar/", directory)

	// Assert
	require.NoError(t, err)
	content, _ := os.ReadFile(filepath)
	require.EqualValues(t, `<base href="/bar/">`, string(content))
}

// Test that the base HTML tag with space before closing curly bracket is
// supported.
func TestSetBaseURLSpaceBeforeCurlyBracket(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()
	filepath, _ := sb.Write("index.html", `<base href="/foo/" >`)
	directory := path.Dir(filepath)

	// Act
	err := setBaseURLInIndexFile("/bar/", directory)

	// Assert
	require.NoError(t, err)
	content, _ := os.ReadFile(filepath)
	require.EqualValues(t, `<base href="/bar/" >`, string(content))
}

// Test that the leading slash is required.
func TestSetBaseURLRequiredLeadingSlash(t *testing.T) {
	// Arrange & Act
	err := setBaseURLInIndexFile("bar/", "")

	// Assert
	require.ErrorContains(t, err, "must start with slash")
}

// Test that the trailing slash is required.
func TestSetBaseURLRequiredTrailingSlash(t *testing.T) {
	// Arrange & Act
	err := setBaseURLInIndexFile("/bar", "")

	// Assert
	require.ErrorContains(t, err, "must end with slash")
}

// Test that no error is returned if the index file doesn't exist.
func TestSetBaseURLForMissingIndexFile(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()

	// Act
	err := setBaseURLInIndexFile("/bar/", sb.BasePath)

	// Assert
	require.NoError(t, err)
}

// Test that no error is returned if there is no read right on the index file.
func TestSetBaseURLForInsufficientReadPermission(t *testing.T) {
	// Arrange
	storktest.SkipIfCurrentUserIgnoresFilePermissions(t)

	sb := testutil.NewSandbox()
	defer sb.Close()

	filepath, _ := sb.Write("index.html", `<base href="/foo/">`)
	directory := path.Dir(filepath)
	_ = os.Chmod(filepath, 0o200)

	// Act
	err := setBaseURLInIndexFile("/bar/", directory)

	// Assert
	require.NoError(t, err)
	_ = os.Chmod(filepath, 0o700)
	content, _ := os.ReadFile(filepath)
	require.EqualValues(t, `<base href="/foo/">`, string(content))
}

// Test that an error is returned if there is no write right on the index file.
func TestSetBaseURLForInsufficientWritePermission(t *testing.T) {
	// Arrange
	storktest.SkipIfCurrentUserIgnoresFilePermissions(t)

	sb := testutil.NewSandbox()
	defer sb.Close()
	filepath, _ := sb.Write("index.html", `<base href="/foo/">`)
	directory := path.Dir(filepath)
	_ = os.Chmod(filepath, 0o400)

	// Act
	err := setBaseURLInIndexFile("/bar/", directory)

	// Assert
	require.Error(t, err)
	_ = os.Chmod(filepath, 0o700)
	content, _ := os.ReadFile(filepath)
	require.EqualValues(t, `<base href="/foo/">`, string(content))
}

// Test that no error is returned if there is no write right on the index file
// but the base tag is not changed.
func TestSetBaseURLForInsufficientWritePermissionExistingValue(t *testing.T) {
	// Arrange
	storktest.SkipIfCurrentUserIgnoresFilePermissions(t)

	sb := testutil.NewSandbox()
	defer sb.Close()
	filepath, _ := sb.Write("index.html", `<base href="/foo/">`)
	directory := path.Dir(filepath)
	_ = os.Chmod(filepath, 0o400)

	// Act
	err := setBaseURLInIndexFile("/foo/", directory)

	// Assert
	require.NoError(t, err)
	_ = os.Chmod(filepath, 0o700)
	content, _ := os.ReadFile(filepath)
	require.EqualValues(t, `<base href="/foo/">`, string(content))
}
