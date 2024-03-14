package prometheus

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/charmbracelet/charm/server/db"
	"github.com/charmbracelet/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Stats contains all of the calls to track metrics.
type Stats struct {
	apiLinkGenCalls     prometheus.Counter
	apiLinkRequestCalls prometheus.Counter
	apiUnlinkCalls      prometheus.Counter
	apiAuthCalls        prometheus.Counter
	apiKeysCalls        prometheus.Counter
	linkGenCalls        prometheus.Counter
	linkRequestCalls    prometheus.Counter
	keysCalls           prometheus.Counter
	idCalls             prometheus.Counter
	jwtCalls            prometheus.Counter
	getUserByIDCalls    prometheus.Counter
	getUserCalls        prometheus.Counter
	setUserNameCalls    prometheus.Counter
	getNews             prometheus.Counter
	postNews            prometheus.Counter
	getNewsList         prometheus.Counter
	fsBytesRead         *prometheus.CounterVec
	fsBytesWritten      *prometheus.CounterVec
	fsReads             *prometheus.CounterVec
	fsWritten           *prometheus.CounterVec
	users               prometheus.Gauge
	userNames           prometheus.Gauge
	db                  db.DB
	port                int
	server              *http.Server
}

// Start starts the PrometheusStats HTTP server.
func (ps *Stats) Start() error {
	// collect totals every minute
	go func() {
		for {
			c, err := ps.db.UserCount()
			if err == nil {
				ps.users.Set(float64(c))
			}
			c, err = ps.db.UserNameCount()
			if err == nil {
				ps.userNames.Set(float64(c))
			}

			time.Sleep(time.Minute)
		}
	}()
	log.Info("Starting Stats HTTP server", "addr", ps.server.Addr)
	err := ps.server.ListenAndServe()
	if err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown shuts down the Stats HTTP server.
func (ps *Stats) Shutdown(ctx context.Context) error {
	return ps.server.Shutdown(ctx)
}

// Close immediately closes the Stats HTTP server.
func (ps *Stats) Close() error {
	return ps.server.Close()
}

// NewStats returns a new Stats HTTP server configured to
// the supplied port.
func NewStats(db db.DB, port int) *Stats {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	fsLabels := []string{"charm_id"}
	return &Stats{
		apiLinkGenCalls:     newCounter("charm_id_api_link_gen_total", "Total API link gen calls"),
		apiLinkRequestCalls: newCounter("charm_id_api_link_request_total", "Total api link request calls"),
		apiUnlinkCalls:      newCounter("charm_id_api_unlink_total", "Total api unlink calls"),
		apiAuthCalls:        newCounter("charm_id_api_auth_total", "Total api auth calls"),
		apiKeysCalls:        newCounter("charm_id_api_keys_total", "Total api keys calls"),
		linkGenCalls:        newCounter("charm_id_link_gen_total", "Total link gen calls"),
		linkRequestCalls:    newCounter("charm_id_link_request_total", "Total link request calls"),
		keysCalls:           newCounter("charm_id_keys_total", "Total keys calls"),
		idCalls:             newCounter("charm_id_id_total", "Total id calls"),
		jwtCalls:            newCounter("charm_id_jwt_total", "Total jwt calls"),
		getUserByIDCalls:    newCounter("charm_bio_get_user_by_id_total", "Total bio user by id calls"),
		getUserCalls:        newCounter("charm_bio_get_user_total", "Total bio get user calls"),
		setUserNameCalls:    newCounter("charm_bio_set_username_total", "Total total bio set username calls"),
		getNews:             newCounter("charm_news_get_news_total", "Total get news calls"),
		postNews:            newCounter("charm_news_post_news_total", "Total post news calls"),
		getNewsList:         newCounter("charm_news_get_news_list_total", "Total get news list calls"),
		fsBytesRead:         newCounterWithLabels("charm_fs_bytes_read_total", "Total bytes read", fsLabels),
		fsBytesWritten:      newCounterWithLabels("charm_fs_bytes_written_total", "Total bytes written", fsLabels),
		fsReads:             newCounterWithLabels("charm_fs_files_read_total", "Total files read", fsLabels),
		fsWritten:           newCounterWithLabels("charm_fs_files_written_total", "Total files read", fsLabels),
		users:               newGauge("charm_bio_users", "Total users"),
		userNames:           newGauge("charm_bio_users_names", "Total usernames"),
		db:                  db,
		port:                port,
		server:              s,
	}
}

// APILinkGen increments the number of api-link-gen calls.
func (ps *Stats) APILinkGen() {
	ps.apiLinkGenCalls.Inc()
}

// APILinkRequest increments the number of api-link-request calls.
func (ps *Stats) APILinkRequest() {
	ps.apiLinkRequestCalls.Inc()
}

// APIUnlink increments the number of api-unlink calls.
func (ps *Stats) APIUnlink() {
	ps.apiUnlinkCalls.Inc()
}

// APIAuth increments the number of api-auth calls.
func (ps *Stats) APIAuth() {
	ps.apiAuthCalls.Inc()
}

// APIKeys increments the number of api-keys calls.
func (ps *Stats) APIKeys() {
	ps.apiKeysCalls.Inc()
}

// LinkGen increments the number of link-gen calls.
func (ps *Stats) LinkGen() {
	ps.linkGenCalls.Inc()
}

// LinkRequest increments the number of link-request calls.
func (ps *Stats) LinkRequest() {
	ps.linkRequestCalls.Inc()
}

// Keys increments the number of keys calls.
func (ps *Stats) Keys() {
	ps.keysCalls.Inc()
}

// ID increments the number of id calls.
func (ps *Stats) ID() {
	ps.idCalls.Inc()
}

// JWT increments the number of jwt calls.
func (ps *Stats) JWT() {
	ps.jwtCalls.Inc()
}

// GetUserByID increments the number of user-by-id calls.
func (ps *Stats) GetUserByID() {
	ps.getUserByIDCalls.Inc()
}

// GetUser increments the number of get-user calls.
func (ps *Stats) GetUser() {
	ps.getUserCalls.Inc()
}

// SetUserName increments the number of set-user-name calls.
func (ps *Stats) SetUserName() {
	ps.setUserNameCalls.Inc()
}

// GetNews increments the number of get-news calls.
func (ps *Stats) GetNews() {
	ps.getNews.Inc()
}

// PostNews increments the number of post-news calls.
func (ps *Stats) PostNews() {
	ps.postNews.Inc()
}

// GetNewsList increments the number of get-news-list calls.
func (ps *Stats) GetNewsList() {
	ps.getNewsList.Inc()
}

// FSFileRead reports metrics on a read file by a given charm_id.
func (ps *Stats) FSFileRead(id string, size int64) {
	ps.fsReads.WithLabelValues(id).Inc()
	ps.fsBytesRead.WithLabelValues(id).Add(float64(size))
}

// FSFileWritten reports metrics on a written file by a given charm_id.
func (ps *Stats) FSFileWritten(id string, size int64) {
	ps.fsWritten.WithLabelValues(id).Inc()
	ps.fsBytesWritten.WithLabelValues(id).Add(float64(size))
}

func newCounter(name string, help string) prometheus.Counter {
	return promauto.NewCounter(prometheus.CounterOpts{
		Name: name,
		Help: help,
	})
}

func newCounterWithLabels(name string, help string, labels []string) *prometheus.CounterVec {
	return promauto.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: help,
	}, labels)
}

func newGauge(name string, help string) prometheus.Gauge {
	return promauto.NewGauge(prometheus.GaugeOpts{
		Name: name,
		Help: help,
	})
}
