package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	storkutil "isc.org/stork/util"
)

// Settings for Prometheus BIND 9 Exporter
type PromBind9ExporterSettings struct {
	Host     string `long:"prometheus-bind9-exporter-host" description:"the IP to listen on" default:"0.0.0.0" env:"STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_ADDRESS"`
	Port     int    `long:"prometheus-bind9-exporter-port" description:"the port to listen on for connections" default:"9548" env:"STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_PORT"`
	Interval int    `long:"prometheus-bind9-exporter-interval" description:"interval of collecting BIND 9 stats in seconds" default:"10" env:"STORK_AGENT_PROMETHEUS_BIND9_EXPORTER_INTERVAL"`
}

// Main structure for Prometheus BIND 9 Exporter. It holds its settings,
// references to app monitor, HTTP client, HTTP server, and main loop
// controlling elements like ticker, and mappings between BIND 9 stats
// names to prometheus stats.
type PromBind9Exporter struct {
	Settings PromBind9ExporterSettings

	AppMonitor AppMonitor
	HTTPClient *HTTPClient
	HTTPServer *http.Server

	Ticker        *time.Ticker
	DoneCollector chan bool
	Wg            *sync.WaitGroup

	CacheStatsMap map[string]*prometheus.GaugeVec
}

// Create new Prometheus BIND 9 Exporter.
func NewPromBind9Exporter(appMonitor AppMonitor) *PromBind9Exporter {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{
		Handler: mux,
	}
	pbe := &PromBind9Exporter{
		AppMonitor:    appMonitor,
		HTTPClient:    NewHTTPClient(),
		HTTPServer:    srv,
		DoneCollector: make(chan bool),
		Wg:            &sync.WaitGroup{},
	}

	// cache stats
	cacheStatsMap := make(map[string]*prometheus.GaugeVec)

	// cache hits
	cacheStatsMap["CacheHits"] = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeBind9,
		Subsystem: "cache",
		Name:      "hits",
		Help:      "Number of cache hits",
	}, []string{"cache"})

	// cache misses
	cacheStatsMap["CacheMisses"] = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeBind9,
		Subsystem: "cache",
		Name:      "misses",
		Help:      "Number of cache misses",
	}, []string{"cache"})

	// cache hit ratio
	cacheStatsMap["CacheHitRatio"] = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: AppTypeBind9,
		Subsystem: "cache",
		Name:      "hit_ratio",
		Help:      "Cache effectiveness (cache hit ratio)",
	}, []string{"cache"})

	pbe.CacheStatsMap = cacheStatsMap

	return pbe
}

// Start goroutine with main loop for collecting stats and HTTP server for
// exposing them to Prometheus.
func (pbe *PromBind9Exporter) Start() {
	// set address for listening from settings
	addrPort := fmt.Sprintf("%s:%d", pbe.Settings.Host, pbe.Settings.Port)
	pbe.HTTPServer.Addr = addrPort

	log.Printf("Prometheus BIND 9 Exporter listening on %s, stats pulling interval: %d seconds", addrPort, pbe.Settings.Interval)

	// start HTTP server for metrics
	go func() {
		err := pbe.HTTPServer.ListenAndServe()
		if err != nil {
			log.Errorf("problem with serving Prometheus BIND 9 Exporter: %s", err.Error())
		}
	}()

	// set ticker time for collecting loop from settings
	pbe.Ticker = time.NewTicker(time.Duration(pbe.Settings.Interval) * time.Second)

	// start collecting loop as goroutine and increment WaitGroup (which is used later
	// for stopping this goroutine)
	pbe.Wg.Add(1)
	go pbe.statsCollectorLoop()
}

// Shutdown exporter goroutines and unregister prometheus stats.
func (pbe *PromBind9Exporter) Shutdown() {
	log.Printf("Stopping Prometheus BIND 9 Exporter")

	// stop http server
	if pbe.HTTPServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		pbe.HTTPServer.SetKeepAlivesEnabled(false)
		if err := pbe.HTTPServer.Shutdown(ctx); err != nil {
			log.Warnf("Could not gracefully shutdown the BIND 9 exporter: %v\n", err)
		}
	}

	// stop stats collector
	if pbe.Ticker != nil {
		pbe.Ticker.Stop()
		pbe.DoneCollector <- true
		pbe.Wg.Wait()
	}

	// unregister bind 9 counters from prometheus framework
	for _, stat := range pbe.CacheStatsMap {
		prometheus.Unregister(stat)
	}

	log.Printf("Stopped Prometheus BIND 9 Exporter")
}

// Main loop for collecting stats periodically.
func (pbe *PromBind9Exporter) statsCollectorLoop() {
	defer pbe.Wg.Done()
	for {
		select {
		// every N seconds do stats collection from all BIND 9 apps
		case <-pbe.Ticker.C:
			err := pbe.collectStats()
			if err != nil {
				log.Errorf("some errors were encountered while collecting stats from BIND 9: %+v", err)
			}
		// wait for done signal from shutdown function
		case <-pbe.DoneCollector:
			return
		}
	}
}

// setDaemonStats stores the stat values from a daemon in the proper prometheus object.
func (pbe *PromBind9Exporter) setDaemonStats(rspIfc interface{}) error {
	rsp, ok := rspIfc.(map[string]interface{})
	if !ok {
		return errors.Errorf("problem with casting rspIfc: %+v", rspIfc)
	}

	// views
	viewsIfc, ok := rsp["views"]
	if !ok {
		return errors.Errorf("no 'views' in response: %+v", rsp)
	}

	views := viewsIfc.(map[string]interface{})
	if !ok {
		return errors.Errorf("problem with casting viewsIfc: %+v", viewsIfc)
	}

	for viewName, viewStatsIfc := range views {
		// Only default view for now.
		if viewName != "_default" {
			continue
		}

		viewStats, ok := viewStatsIfc.(map[string]interface{})
		if !ok {
			log.Errorf("problem with casting viewStatsIfc: %+v", viewStatsIfc)
			continue
		}

		// resolver
		resolverIfc, ok := viewStats["resolver"]
		if !ok {
			log.Errorf("no 'resolver' in viewStats: %+v", viewStats)
			continue
		}
		resolver, ok := resolverIfc.(map[string]interface{})
		if !ok {
			log.Errorf("problem with casting resolverIfc: %+v", resolverIfc)
			continue
		}

		// cachestats
		cachestatsIfc, ok := resolver["cachestats"]
		if !ok {
			log.Errorf("no 'cachestats' in resolver: %+v", resolver)
			continue
		}
		cachestats, ok := cachestatsIfc.(map[string]interface{})
		if !ok {
			log.Errorf("problem with casting cachestatsIfc: %+v", cachestatsIfc)
			continue
		}

		var hit float64
		var miss float64
		for statName, statValueIfc := range cachestats {
			// get stat value
			statValue, ok := statValueIfc.(float64)
			if !ok {
				log.Errorf("problem with casting statValue: %+v", statValueIfc)
				continue
			}
			if statName == "CacheHits" {
				hit = statValue
			} else if statName == "CacheMisses" {
				miss = statValue
			}

			// store stat value in proper prometheus object
			stat, ok := pbe.CacheStatsMap[statName]
			if ok {
				stat.With(prometheus.Labels{"cache": "_default"}).Set(statValue)
			}
		}

		// Set Cache Hit Ratio
		chrStat := pbe.CacheStatsMap["CacheHitRatio"]
		total := hit + miss
		if total > 0 {
			chrStat.With(prometheus.Labels{"cache": "_default"}).Set(hit / total)
		}
	}

	return nil
}

// Collect stats from all BIND 9 apps.
func (pbe *PromBind9Exporter) collectStats() error {
	var lastErr error

	// Request to named statistics-channel for getting all server stats.
	request := `{}`

	// go through all BIND 9 apps discovered by monitor and query them for stats
	apps := pbe.AppMonitor.GetApps()
	for _, app := range apps {
		// ignore non-BIND 9 apps
		if app.Type != AppTypeBind9 {
			continue
		}

		// get stats from named
		sap, err := getAccessPoint(app, AccessPointStatistics)
		if err != nil {
			lastErr = err
			log.Errorf("problem with getting stats from BIND 9, bad access statistics point: %+v", err)
			continue
		}
		address := storkutil.HostWithPortURL(sap.Address, sap.Port)
		path := "json/v1/server"
		url := fmt.Sprintf("%s%s", address, path)
		httpRsp, err := pbe.HTTPClient.Call(url, bytes.NewBuffer([]byte(request)))
		if err != nil {
			lastErr = err
			log.Errorf("problem with getting stats from BIND 9: %+v", err)
			continue
		}
		body, err := ioutil.ReadAll(httpRsp.Body)
		httpRsp.Body.Close()
		if err != nil {
			lastErr = err
			log.Errorf("problem with reading stats response from BIND 9: %+v", err)
			continue
		}

		// parse response
		var rspIfc interface{}
		response := string(body)
		err = json.Unmarshal([]byte(response), &rspIfc)
		if err != nil {
			lastErr = err
			log.Errorf("failed to parse responses from BIND 9: %s", err)
			continue
		}

		err = pbe.setDaemonStats(rspIfc)
		if err != nil {
			log.Errorf("cannot get stat from daemon: %+v", err)
		}
	}

	return lastErr
}
