package agent

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
)

func TestGetApps(t *testing.T) {
	// Forces gock to intercept the HTTP/1.1 client. Otherwise it would
	// use the HTTP/2.
	caClient := NewCAClient()
	gock.InterceptClient(caClient.client)
	sm := NewAppMonitor(caClient)

	apps := sm.GetApps()
	require.Len(t, apps, 0)
	sm.Shutdown()
}

func TestKeaDaemonVersionGetBadUrl(t *testing.T) {
	caClient := NewCAClient()
	gock.InterceptClient(caClient.client)
	_, err := keaDaemonVersionGet(caClient, "aaa", "")
	require.Contains(t, err.Error(), "unsupported protocol ")
}

func TestKeaDaemonVersionGetDataOk(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:45634").
		Post("/").
		Reply(200).
		JSON([]map[string]string{{"arguments": "bar"}})

	caClient := NewCAClient()
	gock.InterceptClient(caClient.client)

	data, err := keaDaemonVersionGet(caClient, "http://localhost:45634/", "")
	require.NoError(t, err)
	require.Equal(t, true, gock.IsDone())
	require.Equal(t, map[string]interface{}{"arguments": "bar"}, data)
}

func TestGetCtrlAddressFromKeaConfigNonExisting(t *testing.T) {
	// check reading from non existing file
	path := "/tmp/non-exisiting-path"
	address, port := getCtrlAddressFromKeaConfig(path)
	require.Equal(t, int64(0), port)
	require.Empty(t, address)
}

func TestGetCtrlFromKeaConfigBadContent(t *testing.T) {
	// prepare kea conf file
	tmpFile, err := ioutil.TempFile(os.TempDir(), "prefix-")
	if err != nil {
		log.Fatal("Cannot create temporary file", err)
	}
	defer os.Remove(tmpFile.Name())

	text := []byte("random content")
	if _, err = tmpFile.Write(text); err != nil {
		log.Fatal("Failed to write to temporary file", err)
	}
	if err := tmpFile.Close(); err != nil {
		log.Fatal(err)
	}

	// check reading from prepared file with bad content
	// so 0 should be returned as port
	address, port := getCtrlAddressFromKeaConfig(tmpFile.Name())
	require.Equal(t, int64(0), port)
	require.Empty(t, address)
}

func TestGetCtrlAddressFromKeaConfigOk(t *testing.T) {
	// prepare kea conf file
	tmpFile, err := ioutil.TempFile(os.TempDir(), "prefix-")
	if err != nil {
		log.Fatal("Cannot create temporary file", err)
	}
	defer os.Remove(tmpFile.Name())

	text := []byte(string("\"http-host\": \"host.example.org\", \"http-port\": 1234"))
	if _, err = tmpFile.Write(text); err != nil {
		log.Fatal("Failed to write to temporary file", err)
	}
	if err := tmpFile.Close(); err != nil {
		log.Fatal(err)
	}

	// check reading from proper file
	address, port := getCtrlAddressFromKeaConfig(tmpFile.Name())
	require.Equal(t, int64(1234), port)
	require.Equal(t, "host.example.org", address)
}

func TestGetCtrlAddressFromKeaConfigAddress0000(t *testing.T) {
	// prepare kea conf file
	tmpFile, err := ioutil.TempFile(os.TempDir(), "prefix-")
	if err != nil {
		log.Fatal("Cannot create temporary file", err)
	}
	defer os.Remove(tmpFile.Name())

	text := []byte(string("\"http-host\": \"0.0.0.0\", \"http-port\": 1234"))
	if _, err = tmpFile.Write(text); err != nil {
		log.Fatal("Failed to write to temporary file", err)
	}
	if err := tmpFile.Close(); err != nil {
		log.Fatal(err)
	}

	// check reading from proper file;
	// if CA is listening on 0.0.0.0 then 127.0.0.1 should be returned
	// as it is not possible to connect to 0.0.0.0
	address, port := getCtrlAddressFromKeaConfig(tmpFile.Name())
	require.Equal(t, int64(1234), port)
	require.Equal(t, "127.0.0.1", address)
}

func TestDetectApps(t *testing.T) {
	caClient := NewCAClient()
	gock.InterceptClient(caClient.client)
	sm := NewAppMonitor(caClient)
	sm.(*appMonitor).detectApps()
	sm.Shutdown()
}

func TestDetectBind9App(t *testing.T) {
	// check bind9 app detection
	srv := detectBind9App()
	require.NotNil(t, srv)
	require.Empty(t, srv.Version)
	require.False(t, srv.Active)
	require.Equal(t, "named", srv.Daemon.Name)
	require.Empty(t, srv.Daemon.Version)
	require.False(t, srv.Daemon.Active)
}

func TestDetectKeaApp(t *testing.T) {
	// prepare kea conf file
	tmpFile, err := ioutil.TempFile(os.TempDir(), "prefix-")
	if err != nil {
		log.Fatal("Cannot create temporary file", err)
	}
	defer os.Remove(tmpFile.Name())

	text := []byte("\"http-host\": \"localhost\", \"http-port\": 45634")
	if _, err = tmpFile.Write(text); err != nil {
		log.Fatal("Failed to write to temporary file", err)
	}
	if err := tmpFile.Close(); err != nil {
		log.Fatal(err)
	}

	// prepare response for ctrl-agent
	defer gock.Off()
	// first request to the kea ctrl-agent
	gock.New("http://localhost:45634").
		Post("/").
		Reply(200).
		JSON([]map[string]interface{}{{
			"arguments": map[string]interface{}{"extended": "bla bla"},
			"result":    0, "text": "1.2.3",
		}})
	// - second request to kea daemon
	gock.New("http://localhost:45634").
		Post("/").
		Reply(200).
		JSON([]map[string]interface{}{{
			"arguments": map[string]interface{}{"extended": "bla bla"},
			"result":    0, "text": "1.2.3",
		}})

	caClient := NewCAClient()
	gock.InterceptClient(caClient.client)

	// check kea app detection
	srv := detectKeaApp(caClient, []string{"", tmpFile.Name()})
	require.Nil(t, srv)
}
