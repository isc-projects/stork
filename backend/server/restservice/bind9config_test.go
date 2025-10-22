package restservice

import (
	context "context"
	"fmt"
	http "net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	bind9config "isc.org/stork/appcfg/bind9"
	"isc.org/stork/server/agentcomm"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/restapi/operations/services"
)

// Test that BIND 9 configuration is returned for different filter combinations.
func TestGetBind9RawConfig(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	mockManager := NewMockManager(ctrl)
	mockManager.EXPECT().GetBind9RawConfig(gomock.Any(), int64(123), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(ctx context.Context, daemonID int64, fileSelector *bind9config.FileTypeSelector, filter *bind9config.Filter) (*agentcomm.Bind9RawConfig, error) {
		// Mock returning different configurations based on the
		// selected config and view filters.
		if filter == nil {
			return &agentcomm.Bind9RawConfig{
				Files: []*agentcomm.Bind9ConfigFile{
					{
						FileType:   agentcomm.Bind9ConfigFileTypeConfig,
						SourcePath: "named.conf",
						Contents:   "config;view;",
					},
				},
			}, nil
		}
		builder := strings.Builder{}
		if filter.IsEnabled(bind9config.FilterTypeConfig) {
			builder.WriteString("config;")
		}
		if filter.IsEnabled(bind9config.FilterTypeView) {
			builder.WriteString("view;")
		}
		return &agentcomm.Bind9RawConfig{
			Files: []*agentcomm.Bind9ConfigFile{
				{
					FileType:   agentcomm.Bind9ConfigFileTypeConfig,
					SourcePath: "named.conf",
					Contents:   builder.String(),
				},
			},
		}, nil
	})

	rapi, err := NewRestAPI(dbSettings, db, mockManager)
	require.NoError(t, err)
	require.NotNil(t, rapi)

	t.Run("no filter", func(t *testing.T) {
		params := services.GetBind9RawConfigParams{
			ID: 123,
		}
		rsp := rapi.GetBind9RawConfig(context.Background(), params)
		require.IsType(t, &services.GetBind9RawConfigOK{}, rsp)
		okRsp := rsp.(*services.GetBind9RawConfigOK)
		require.Len(t, okRsp.Payload.Files, 1)
		require.Equal(t, `config;view;`, okRsp.Payload.Files[0].Contents)
	})

	t.Run("empty filter", func(t *testing.T) {
		params := services.GetBind9RawConfigParams{
			ID:     123,
			Filter: []string{},
		}
		rsp := rapi.GetBind9RawConfig(context.Background(), params)
		require.IsType(t, &services.GetBind9RawConfigOK{}, rsp)
		okRsp := rsp.(*services.GetBind9RawConfigOK)
		require.Len(t, okRsp.Payload.Files, 1)
		require.Equal(t, `config;view;`, okRsp.Payload.Files[0].Contents)
	})

	t.Run("filter config", func(t *testing.T) {
		params := services.GetBind9RawConfigParams{
			ID:     123,
			Filter: []string{"config"},
		}
		rsp := rapi.GetBind9RawConfig(context.Background(), params)
		require.IsType(t, &services.GetBind9RawConfigOK{}, rsp)
		okRsp := rsp.(*services.GetBind9RawConfigOK)
		require.Len(t, okRsp.Payload.Files, 1)
		require.Equal(t, `config;`, okRsp.Payload.Files[0].Contents)
	})

	t.Run("filter view", func(t *testing.T) {
		params := services.GetBind9RawConfigParams{
			ID:     123,
			Filter: []string{"view"},
		}
		rsp := rapi.GetBind9RawConfig(context.Background(), params)
		require.IsType(t, &services.GetBind9RawConfigOK{}, rsp)
		okRsp := rsp.(*services.GetBind9RawConfigOK)
		require.Len(t, okRsp.Payload.Files, 1)
		require.Equal(t, `view;`, okRsp.Payload.Files[0].Contents)
	})

	t.Run("filter config and view", func(t *testing.T) {
		params := services.GetBind9RawConfigParams{
			ID:     123,
			Filter: []string{"config", "view"},
		}
		rsp := rapi.GetBind9RawConfig(context.Background(), params)
		require.IsType(t, &services.GetBind9RawConfigOK{}, rsp)
		okRsp := rsp.(*services.GetBind9RawConfigOK)
		require.Len(t, okRsp.Payload.Files, 1)
		require.Equal(t, `config;view;`, okRsp.Payload.Files[0].Contents)
	})
}

// Test that multiple files are returned when getting BIND 9 configuration
// from the agent.
func TestGetBind9RawConfigMultipleFiles(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	mockManager := NewMockManager(ctrl)
	mockManager.EXPECT().GetBind9RawConfig(gomock.Any(), int64(123), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(ctx context.Context, daemonID int64, fileSelector *bind9config.FileTypeSelector, filter *bind9config.Filter) (*agentcomm.Bind9RawConfig, error) {
		var files []*agentcomm.Bind9ConfigFile
		for _, fileType := range []agentcomm.Bind9ConfigFileType{agentcomm.Bind9ConfigFileTypeConfig, agentcomm.Bind9ConfigFileTypeRndcKey} {
			if fileSelector.IsEnabled(bind9config.FileType(fileType)) {
				files = append(files, &agentcomm.Bind9ConfigFile{
					FileType:   fileType,
					SourcePath: fmt.Sprintf("%s.conf", fileType),
					Contents:   fmt.Sprintf("%s;", fileType),
				})
			}
		}
		return &agentcomm.Bind9RawConfig{
			Files: files,
		}, nil
	})

	rapi, err := NewRestAPI(dbSettings, db, mockManager)
	require.NoError(t, err)
	require.NotNil(t, rapi)

	t.Run("no file selection", func(t *testing.T) {
		params := services.GetBind9RawConfigParams{
			ID: 123,
		}
		rsp := rapi.GetBind9RawConfig(context.Background(), params)
		require.IsType(t, &services.GetBind9RawConfigOK{}, rsp)
		okRsp := rsp.(*services.GetBind9RawConfigOK)
		require.Len(t, okRsp.Payload.Files, 2)
		require.Equal(t, `config;`, okRsp.Payload.Files[0].Contents)
		require.Equal(t, "config.conf", okRsp.Payload.Files[0].SourcePath)
		require.Equal(t, string(agentcomm.Bind9ConfigFileTypeConfig), okRsp.Payload.Files[0].FileType)
		require.Equal(t, `rndc-key;`, okRsp.Payload.Files[1].Contents)
		require.Equal(t, "rndc-key.conf", okRsp.Payload.Files[1].SourcePath)
		require.Equal(t, string(agentcomm.Bind9ConfigFileTypeRndcKey), okRsp.Payload.Files[1].FileType)
	})

	t.Run("select only config file", func(t *testing.T) {
		params := services.GetBind9RawConfigParams{
			ID:           123,
			FileSelector: []string{"config"},
		}
		rsp := rapi.GetBind9RawConfig(context.Background(), params)
		require.IsType(t, &services.GetBind9RawConfigOK{}, rsp)
		okRsp := rsp.(*services.GetBind9RawConfigOK)
		require.Len(t, okRsp.Payload.Files, 1)
		require.Equal(t, `config;`, okRsp.Payload.Files[0].Contents)
	})

	t.Run("select only rndc key file", func(t *testing.T) {
		params := services.GetBind9RawConfigParams{
			ID:           123,
			FileSelector: []string{"rndc-key"},
		}
		rsp := rapi.GetBind9RawConfig(context.Background(), params)
		require.IsType(t, &services.GetBind9RawConfigOK{}, rsp)
		okRsp := rsp.(*services.GetBind9RawConfigOK)
		require.Len(t, okRsp.Payload.Files, 1)
		require.Equal(t, `rndc-key;`, okRsp.Payload.Files[0].Contents)
	})

	t.Run("select both config and rndc key files", func(t *testing.T) {
		params := services.GetBind9RawConfigParams{
			ID:           123,
			FileSelector: []string{"config", "rndc-key"},
		}
		rsp := rapi.GetBind9RawConfig(context.Background(), params)
		require.IsType(t, &services.GetBind9RawConfigOK{}, rsp)
		okRsp := rsp.(*services.GetBind9RawConfigOK)
		require.Len(t, okRsp.Payload.Files, 2)
		require.Equal(t, `config;`, okRsp.Payload.Files[0].Contents)
		require.Equal(t, `rndc-key;`, okRsp.Payload.Files[1].Contents)
	})
}

// Test that an error is returned when getting BIND 9 configuration fails.
func TestGetBind9RawConfigError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	mockManager := NewMockManager(ctrl)
	mockManager.EXPECT().GetBind9RawConfig(gomock.Any(), int64(123), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, daemonID int64, fileSelector *bind9config.FileTypeSelector, filter *bind9config.Filter) (*agentcomm.Bind9RawConfig, error) {
		return nil, &testError{}
	})

	rapi, err := NewRestAPI(dbSettings, db, mockManager)
	require.NoError(t, err)
	require.NotNil(t, rapi)

	params := services.GetBind9RawConfigParams{
		ID: 123,
	}
	rsp := rapi.GetBind9RawConfig(context.Background(), params)
	require.IsType(t, &services.GetBind9RawConfigDefault{}, rsp)
	defaultRsp := rsp.(*services.GetBind9RawConfigDefault)
	require.Equal(t, http.StatusInternalServerError, getStatusCode(*defaultRsp))
	require.Equal(t, "Cannot get BIND 9 configuration for daemon with ID 123", *defaultRsp.Payload.Message)
}
