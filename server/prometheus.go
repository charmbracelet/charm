package server

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PrometheusStats struct {
	APILinkGenCalls     prometheus.Counter
	APILinkRequestCalls prometheus.Counter
	APIUnlinkCalls      prometheus.Counter
	APIAuthCalls        prometheus.Counter
	APIKeysCalls        prometheus.Counter
	LinkGenCalls        prometheus.Counter
	LinkRequestCalls    prometheus.Counter
	KeysCalls           prometheus.Counter
	IDCalls             prometheus.Counter
	JWTCalls            prometheus.Counter
	GetUserByIDCalls    prometheus.Counter
	GetUserCalls        prometheus.Counter
	SetUserNameCalls    prometheus.Counter
	Users               prometheus.Gauge
	UserNames           prometheus.Gauge
	db                  DB
	port                int
}

func (ps *PrometheusStats) Start() {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", ps.port),
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	ps.collectStats()
	log.Fatal(s.ListenAndServe())
}

func NewPrometheusStats(db DB, port int) PrometheusStats {
	return PrometheusStats{
		APILinkGenCalls:     newCounter("charm_id_api_link_gen_total", "Total API link gen calls"),
		APILinkRequestCalls: newCounter("charm_id_api_link_request_total", "Total API link request calls"),
		APIUnlinkCalls:      newCounter("charm_id_api_unlink_total", "Total API unlink calls"),
		APIAuthCalls:        newCounter("charm_id_api_auth_total", "Total API auth calls"),
		APIKeysCalls:        newCounter("charm_id_api_keys_total", "Total API keys calls"),
		LinkGenCalls:        newCounter("charm_id_link_gen_total", "Total link gen calls"),
		LinkRequestCalls:    newCounter("charm_id_link_request_total", "Total link request calls"),
		KeysCalls:           newCounter("charm_id_keys_total", "Total keys calls"),
		IDCalls:             newCounter("charm_id_id_total", "Total id calls"),
		JWTCalls:            newCounter("charm_id_jwt_total", "Total jwt calls"),
		GetUserByIDCalls:    newCounter("charm_bio_get_user_by_id_total", "Total bio user by id calls"),
		GetUserCalls:        newCounter("charm_bio_get_user_total", "Total bio get user calls"),
		SetUserNameCalls:    newCounter("charm_bio_set_username_total", "Total total bio set username calls"),
		Users:               newGauge("charm_bio_users", "Total users"),
		UserNames:           newGauge("charm_bio_users_names", "Total usernames"),
		db:                  db,
		port:                port,
	}
}

func newCounter(name string, help string) prometheus.Counter {
	return promauto.NewCounter(prometheus.CounterOpts{
		Name: name,
		Help: help,
	})
}

func newGauge(name string, help string) prometheus.Gauge {
	return promauto.NewGauge(prometheus.GaugeOpts{
		Name: name,
		Help: help,
	})
}

func (ps PrometheusStats) collectStats() {
	go func() {
		for {
			c, err := ps.db.UserCount()
			if err == nil {
				ps.Users.Set(float64(c))
			}
			c, err = ps.db.UserNameCount()
			if err == nil {
				ps.UserNames.Set(float64(c))
			}

			time.Sleep(time.Minute)
		}
	}()
}
