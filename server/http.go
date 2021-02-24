package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/charmbracelet/charm"
	"github.com/meowgorithm/babylogger"
	"goji.io"
	"goji.io/pat"
)

type HTTPServer struct {
	db     DB
	fstore FileStore
	stats  PrometheusStats
	cfg    Config
	mux    *goji.Mux
}

type JSONError struct {
	Message string `json:"message"`
}

func NewHTTPServer(cfg Config) *HTTPServer {
	// No auth health check endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "We live!")
	})
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%s", cfg.HealthPort), nil)
		if err != nil {
			log.Fatalf("http server exited with error: %s", err)
		}
	}()

	mux := goji.NewMux()
	s := &HTTPServer{
		cfg: cfg,
		mux: mux,
	}

	var charmMiddleware func(http.Handler) http.Handler
	charmMiddleware, err := JWTMiddleware(cfg.PublicKey)
	if err != nil {
		log.Fatalf("could not create jwt middleware: %s", err)
	}
	mux.Use(babylogger.Middleware)
	mux.Use(charmMiddleware)
	mux.Use(stripTrailingSlashMiddleware)
	mux.HandleFunc(pat.Get("/v1/id/:id"), s.handleGetUserByID)
	mux.HandleFunc(pat.Get("/v1/bio/:name"), s.handleGetUser)
	mux.HandleFunc(pat.Post("/v1/bio"), s.handlePostUser)
	mux.HandleFunc(pat.Post("/v1/encrypt-key"), s.handlePostEncryptKey)
	mux.HandleFunc(pat.Get("/v1/datalog/:name/:seq"), s.handleGetDatalogSeq)
	mux.HandleFunc(pat.Post("/v1/datalog/:name/:seq"), s.handlePostDatalogSeq)
	mux.HandleFunc(pat.Get("/v1/seq/:name"), s.handleGetSeq)
	mux.HandleFunc(pat.Post("/v1/seq/:name"), s.handlePostSeq)
	s.db = cfg.DB
	s.fstore = cfg.FileStore
	return s
}

func (s *HTTPServer) renderError(w http.ResponseWriter) {
	s.renderCustomError(w, "internal error", http.StatusInternalServerError)
}

func (s *HTTPServer) renderCustomError(w http.ResponseWriter, msg string, status int) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(JSONError{msg})
}

func (s *HTTPServer) handleGetUserByID(w http.ResponseWriter, r *http.Request) {
	id := pat.Param(r, "id")
	u, err := s.db.GetUserWithID(id)
	if err == charm.ErrMissingUser {
		s.renderCustomError(w, fmt.Sprintf("missing user for id '%s'", id), http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("cannot read request body: %s", err)
		s.renderError(w)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(u)
	// s.stats.GetUserByIDCalls.Inc()
}

func (s *HTTPServer) handleGetUser(w http.ResponseWriter, r *http.Request) {
	name := pat.Param(r, "name")
	u, err := s.db.GetUserWithName(name)
	if err == charm.ErrMissingUser {
		s.renderCustomError(w, fmt.Sprintf("missing user for name '%s'", name), http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("cannot read request body: %s", err)
		s.renderError(w)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(u)
	// s.stats.GetUserCalls.Inc()
}

func (s *HTTPServer) handlePostUser(w http.ResponseWriter, r *http.Request) {
	id, err := CharmIdFromRequest(r)
	if err != nil {
		log.Printf("cannot read request body: %s", err)
		s.renderError(w)
		return
	}
	u := &charm.User{}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("cannot read request body: %s", err)
		s.renderError(w)
		return
	}
	err = json.Unmarshal(body, u)
	if err != nil {
		log.Printf("cannot decode user json: %s", err)
		s.renderError(w)
		return
	}
	nu, err := s.db.SetUserName(id, u.Name)
	if err == charm.ErrNameTaken {
		s.renderCustomError(w, fmt.Sprintf("username '%s' already taken", u.Name), http.StatusConflict)
	} else if err != nil {
		log.Printf("cannot set user name: %s", err)
		s.renderError(w)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(nu)
	// s.stats.SetUserNameCalls.Inc()
}

func (s *HTTPServer) handlePostEncryptKey(w http.ResponseWriter, r *http.Request) {
	id, err := CharmIdFromRequest(r)
	if err != nil {
		log.Printf("cannot read request body: %s", err)
		s.renderError(w)
		return
	}
	u, err := s.db.GetUserWithID(id)
	if err != nil {
		log.Printf("cannot fetch user: %s", err)
		s.renderError(w)
		return
	}
	ek := &charm.EncryptKey{}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("cannot read request body: %s", err)
		s.renderError(w)
		return
	}
	err = json.Unmarshal(body, ek)
	if err != nil {
		log.Printf("cannot decode encrypt key json: %s", err)
		s.renderError(w)
		return
	}
	err = s.db.AddEncryptKeyForPublicKey(u, ek.PublicKey, ek.GlobalID, ek.Key)
	if err != nil {
		log.Printf("cannot add encrypt key: %s", err)
		s.renderError(w)
		return
	}
	// s.stats.SetUserNameCalls.Inc()
}

func (s *HTTPServer) handleGetSeq(w http.ResponseWriter, r *http.Request) {
	name := pat.Param(r, "name")
	id, err := CharmIdFromRequest(r)
	if err != nil {
		log.Printf("cannot read request body: %s", err)
		s.renderError(w)
		return
	}
	u, err := s.db.GetUserWithID(id)
	if err != nil {
		log.Printf("cannot fetch user: %s", err)
		s.renderError(w)
		return
	}
	seq, err := s.db.GetSeq(u, name)
	if err != nil {
		log.Printf("cannot get seq: %s", err)
		s.renderError(w)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(&charm.SeqMsg{Seq: seq})
}

func (s *HTTPServer) handlePostSeq(w http.ResponseWriter, r *http.Request) {
	name := pat.Param(r, "name")
	id, err := CharmIdFromRequest(r)
	if err != nil {
		log.Printf("cannot read request body: %s", err)
		s.renderError(w)
		return
	}
	u, err := s.db.GetUserWithID(id)
	if err != nil {
		log.Printf("cannot fetch user: %s", err)
		s.renderError(w)
		return
	}
	seq, err := s.db.NextSeq(u, name)
	if err != nil {
		log.Printf("cannot get next seq: %s", err)
		s.renderError(w)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(&charm.SeqMsg{Seq: seq})
}

func (s *HTTPServer) handlePostDatalogSeq(w http.ResponseWriter, r *http.Request) {
	name := pat.Param(r, "name")
	seq := pat.Param(r, "seq")
	_, err := strconv.Atoi(seq)
	if err != nil {
		log.Printf("seq not a number: %s", err)
		s.renderError(w)
		return
	}
	id, err := CharmIdFromRequest(r)
	if err != nil {
		log.Printf("cannot read request body: %s", err)
		s.renderError(w)
		return
	}
	_, err = s.db.GetUserWithID(id)
	if err != nil {
		log.Printf("cannot fetch user: %s", err)
		s.renderError(w)
		return
	}
	f, _, err := r.FormFile("data")
	if err != nil {
		log.Printf("cannot parse form data: %s", err)
		s.renderError(w)
		return
	}
	defer f.Close()
	fk := strings.Join([]string{name, seq}, "/")
	err = s.cfg.FileStore.Put(id, fk, f)
	if err != nil {
		log.Printf("cannot store datalog file: %s", err)
		s.renderError(w)
		return
	}
}

func (s *HTTPServer) handleGetDatalogSeq(w http.ResponseWriter, r *http.Request) {
	name := pat.Param(r, "name")
	seq := pat.Param(r, "seq")
	_, err := strconv.Atoi(seq)
	if err != nil {
		log.Printf("seq not a number: %s", err)
		s.renderError(w)
		return
	}
	id, err := CharmIdFromRequest(r)
	if err != nil {
		log.Printf("cannot read request body: %s", err)
		s.renderError(w)
		return
	}
	_, err = s.db.GetUserWithID(id)
	if err != nil {
		log.Printf("cannot fetch user: %s", err)
		s.renderError(w)
		return
	}
	fk := strings.Join([]string{name, seq}, "/")
	err = s.cfg.FileStore.Get(id, fk, w)
	if err != nil {
		log.Printf("cannot get datalog file: %s", err)
		s.renderError(w)
		return
	}
}

func (s *HTTPServer) Start() {
	listenAddr := fmt.Sprintf(":%d", s.cfg.HTTPPort)
	log.Printf("HTTP server listening on: %s", listenAddr)
	log.Fatalf("Server crashed: %s", http.ListenAndServe(listenAddr, s.mux))
}

func stripTrailingSlashMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		i := strings.LastIndex(p, "/")
		if len(p) > 1 && i == len(p)-1 {
			http.Redirect(w, r, p[:len(p)-1], http.StatusMovedPermanently)
			return
		}
		next.ServeHTTP(w, r)
	})
}
