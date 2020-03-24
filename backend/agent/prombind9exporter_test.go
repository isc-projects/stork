package agent

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
)

// Fake app monitor that returns some predefined list of apps.
type PromFakeBind9AppMonitor struct {
	Apps []*App
}

func (fam *PromFakeBind9AppMonitor) GetApps() []*App {
	log.Println("GetApps")
	accessPoints := makeAccessPoint(AccessPointStatistics, "1.2.3.4", "", 1234)
	accessPoints = append(accessPoints, AccessPoint{
		Type:    AccessPointControl,
		Address: "1.9.5.3",
		Port:    1953,
		Key:     "abcd",
	})
	return []*App{{
		Type:         AppTypeBind9,
		AccessPoints: accessPoints,
	}}
}

func (fam *PromFakeBind9AppMonitor) Shutdown() {
}

// Check creating PromBind9Exporter, check if prometheus stats are set up.
func TestNewPromBind9ExporterBasic(t *testing.T) {
	fam := &PromFakeBind9AppMonitor{}
	pke := NewPromBind9Exporter(fam)
	defer pke.Shutdown()

	require.NotNil(t, pke.HTTPClient)
	require.NotNil(t, pke.HTTPServer)
	require.Len(t, pke.CacheStatsMap, 3)
}

// Check starting PromBind9Exporter and collecting stats.
func TestPromBind9ExporterStart(t *testing.T) {
	defer gock.Off()
	gock.New("http://1.2.3.4:1234/").
		Post("/").
		Persist().
		Reply(200).
		BodyString(`{"views": { "_default":
                            { "resolver": { "cachestats":
                            { "CacheHits": 40, "CacheMisses": 10
                            }}}}}`)
	fam := &PromFakeBind9AppMonitor{}
	pke := NewPromBind9Exporter(fam)
	defer pke.Shutdown()

	gock.InterceptClient(pke.HTTPClient.client)

	// prepare sane settings
	pke.Settings.Port = 1234
	pke.Settings.Interval = 1 // 1 second

	// start exporter
	pke.Start()
	require.NotNil(t, pke.Ticker)

	// wait 1.5 seconds that collecting is invoked at least once
	time.Sleep(1500 * time.Millisecond)

	// check if cache hits is 40
	metric, _ := pke.CacheStatsMap["CacheHits"].GetMetricWith(prometheus.Labels{"cache": "_default"})
	require.Equal(t, 40.0, testutil.ToFloat64(metric))
	// check if cache hits is 10
	metric, _ = pke.CacheStatsMap["CacheMisses"].GetMetricWith(prometheus.Labels{"cache": "_default"})
	require.Equal(t, 10.0, testutil.ToFloat64(metric))
	// check if cache hit ratio is 0.8
	metric, _ = pke.CacheStatsMap["CacheHitRatio"].GetMetricWith(prometheus.Labels{"cache": "_default"})
	require.Equal(t, 0.8, testutil.ToFloat64(metric))
}
