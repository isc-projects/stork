package agent

import (
	"os"
	"log"
	"io/ioutil"
	"testing"
	"gopkg.in/h2non/gock.v1"
	"github.com/stretchr/testify/require"
)

// Forces gock to intercept the HTTP/1.1 client. Otherwise it would
// use the HTTP/2.
func TestMain(m *testing.M) {
	gock.InterceptClient(httpClient11)
	os.Exit(m.Run())
}

func TestGetApps(t *testing.T) {
	sm := NewAppMonitor()
	apps := sm.GetApps()
	require.Len(t, apps, 0)
	sm.Shutdown()
}

func TestKeaDaemonVersionGetBadUrl(t *testing.T) {
	_, err := keaDaemonVersionGet("aaa", "")
	require.Contains(t, err.Error(), "unsupported protocol ")
}

func TestKeaDaemonVersionGetDataOk(t *testing.T) {
	defer gock.Off()

	gock.New("http://localhost:45634").
		Post("/").
		Reply(200).
		JSON([]map[string]string{{"arguments": "bar"}})

	data, err := keaDaemonVersionGet("http://localhost:45634/", "")
	require.NoError(t, err)
	require.Equal(t, true, gock.IsDone())
	require.Equal(t, map[string]interface{}{"arguments":"bar"}, data)
}

func TestGetCtrlPortFromKeaConfigNonExisting(t *testing.T) {
	// check reading from non existing file
	path := "/tmp/non-exisiting-path"
	address, port := getCtrlFromKeaConfig(path)
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
	address, port := getCtrlFromKeaConfig(tmpFile.Name())
	require.Equal(t, int64(0), port)
	require.Empty(t, address)
}

func TestGetCtrlPortFromKeaConfigOk(t *testing.T) {
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
	address, port := getCtrlFromKeaConfig(tmpFile.Name())
	require.Equal(t, int64(1234), port)
	require.Equal(t, "host.example.org", address)
}

func TestDetectApps(t *testing.T) {
	sm := NewAppMonitor()
	sm.detectApps()
	sm.Shutdown()
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
			"result": 0, "text": "1.2.3",
		}})
	// - second request to kea daemon
	gock.New("http://localhost:45634").
		Post("/").
		Reply(200).
		JSON([]map[string]interface{}{{
			"arguments": map[string]interface{}{"extended": "bla bla"},
			"result": 0, "text": "1.2.3",
		}})

	// check kea app detection
	srv := detectKeaApp([]string{"", tmpFile.Name()})
	require.Nil(t, srv)
}
