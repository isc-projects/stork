package restservice

import (
	context "context"
	iter "iter"
	http "net/http"
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	agentapi "isc.org/stork/api"
	bind9config "isc.org/stork/appcfg/bind9"
	dbtest "isc.org/stork/server/database/test"
	dnsop "isc.org/stork/server/dnsop"
	"isc.org/stork/server/gen/restapi/operations/services"
	storkutil "isc.org/stork/util"
)

// Test successfully getting BIND 9 configuration.
func TestGetBind9FormattedConfig(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	mockManager := NewMockManager(ctrl)
	mockManager.EXPECT().GetBind9FormattedConfig(gomock.Any(), int64(123), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(ctx context.Context, daemonID int64, fileSelector *bind9config.FileTypeSelector, filter *bind9config.Filter) iter.Seq[*dnsop.Bind9FormattedConfigResponse] {
		require.NotNil(t, filter)
		require.True(t, filter.IsEnabled(bind9config.FilterTypeConfig))
		require.True(t, filter.IsEnabled(bind9config.FilterTypeView))
		require.False(t, filter.IsEnabled(bind9config.FilterTypeZone))
		require.NotNil(t, fileSelector)
		require.True(t, fileSelector.IsEnabled(bind9config.FileTypeConfig))
		require.True(t, fileSelector.IsEnabled(bind9config.FileTypeRndcKey))
		return func(yield func(*dnsop.Bind9FormattedConfigResponse) bool) {
			responses := []*dnsop.Bind9FormattedConfigResponse{
				{
					File: &agentapi.ReceiveBind9ConfigFile{
						FileType:   agentapi.Bind9ConfigFileType_CONFIG,
						SourcePath: "named.conf",
					},
				},
				{
					Contents: storkutil.Ptr("config;"),
				},
				{
					Contents: storkutil.Ptr("view;"),
				},
				{
					File: &agentapi.ReceiveBind9ConfigFile{
						FileType:   agentapi.Bind9ConfigFileType_RNDC_KEY,
						SourcePath: "rndc.key",
					},
				},
				{
					Contents: storkutil.Ptr("rndc-key;"),
				},
			}
			for _, response := range responses {
				if !yield(response) {
					return
				}
			}
		}
	})
	rapi, err := NewRestAPI(dbSettings, db, mockManager)
	require.NoError(t, err)
	require.NotNil(t, rapi)

	params := services.GetBind9FormattedConfigParams{
		ID:           123,
		Filter:       []string{"config", "view"},
		FileSelector: []string{"config", "rndc-key"},
	}
	rsp := rapi.GetBind9FormattedConfig(context.Background(), params)
	require.IsType(t, &services.GetBind9FormattedConfigOK{}, rsp)
	okRsp := rsp.(*services.GetBind9FormattedConfigOK)
	require.Len(t, okRsp.Payload.Files, 2)
	require.Equal(t, []string{"config;", "view;"}, okRsp.Payload.Files[0].Contents)
	require.Equal(t, []string{"rndc-key;"}, okRsp.Payload.Files[1].Contents)
}

// Test that an error is returned when getting BIND 9 configuration fails.
func TestGetBind9FormattedConfigError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	mockManager := NewMockManager(ctrl)
	mockManager.EXPECT().GetBind9FormattedConfig(gomock.Any(), int64(123), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, daemonID int64, fileSelector *bind9config.FileTypeSelector, filter *bind9config.Filter) iter.Seq[*dnsop.Bind9FormattedConfigResponse] {
		require.Nil(t, filter)
		require.Nil(t, fileSelector)
		return func(yield func(*dnsop.Bind9FormattedConfigResponse) bool) {
			yield(dnsop.NewBind9FormattedConfigResponseError(&testError{}))
		}
	})

	rapi, err := NewRestAPI(dbSettings, db, mockManager)
	require.NoError(t, err)
	require.NotNil(t, rapi)

	params := services.GetBind9FormattedConfigParams{
		ID: 123,
	}
	rsp := rapi.GetBind9FormattedConfig(context.Background(), params)
	require.IsType(t, &services.GetBind9FormattedConfigDefault{}, rsp)
	defaultRsp := rsp.(*services.GetBind9FormattedConfigDefault)
	require.Equal(t, http.StatusInternalServerError, getStatusCode(*defaultRsp))
	require.Equal(t, "Cannot get BIND 9 configuration for daemon with ID 123", *defaultRsp.Payload.Message)
}
