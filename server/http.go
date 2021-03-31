package server

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	charm "github.com/charmbracelet/charm/proto"
	"github.com/meowgorithm/babylogger"
	"goji.io"
	"goji.io/pat"
	"goji.io/pattern"
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

	var jwtMiddleware func(http.Handler) http.Handler
	jwtMiddleware, err := JWTMiddleware(cfg.PublicKey)
	if err != nil {
		log.Fatalf("could not create jwt middleware: %s", err)
	}
	mux.Use(babylogger.Middleware)
	mux.Use(jwtMiddleware)
	mux.Use(CharmUserMiddleware(s))
	mux.HandleFunc(pat.Get("/v1/id/:id"), s.handleGetUserByID)
	mux.HandleFunc(pat.Get("/v1/bio/:name"), s.handleGetUser)
	mux.HandleFunc(pat.Post("/v1/bio"), s.handlePostUser)
	mux.HandleFunc(pat.Post("/v1/encrypt-key"), s.handlePostEncryptKey)
	mux.HandleFunc(pat.Get("/v1/fs/*"), s.handleGetFile)
	mux.HandleFunc(pat.Post("/v1/fs/*"), s.handlePostFile)
	mux.HandleFunc(pat.Delete("/v1/fs/*"), s.handleDeleteFile)
	mux.HandleFunc(pat.Get("/v1/seq/:name"), s.handleGetSeq)
	mux.HandleFunc(pat.Post("/v1/seq/:name"), s.handlePostSeq)
	s.db = cfg.DB
	s.fstore = cfg.FileStore
	return s
}

func (s *HTTPServer) Start() {
	listenAddr := fmt.Sprintf(":%d", s.cfg.HTTPPort)
	log.Printf("HTTP server listening on: %s", listenAddr)
	log.Fatalf("Server crashed: %s", http.ListenAndServe(listenAddr, s.mux))
}

func (s *HTTPServer) renderError(w http.ResponseWriter) {
	s.renderCustomError(w, "internal error", http.StatusInternalServerError)
}

func (s *HTTPServer) renderCustomError(w http.ResponseWriter, msg string, status int) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(JSONError{msg})
}

// TODO do we need this since you can only get the authed user?
func (s *HTTPServer) handleGetUserByID(w http.ResponseWriter, r *http.Request) {
	u := s.charmUserFromRequest(w, r)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(u)
	// s.stats.GetUserByIDCalls.Inc()
}

// TODO do we need this since you can only get the authed user?
func (s *HTTPServer) handleGetUser(w http.ResponseWriter, r *http.Request) {
	u := s.charmUserFromRequest(w, r)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(u)
	// s.stats.GetUserCalls.Inc()
}

func (s *HTTPServer) handlePostUser(w http.ResponseWriter, r *http.Request) {
	id, err := charmIdFromRequest(r)
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
	u := s.charmUserFromRequest(w, r)
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
	u := s.charmUserFromRequest(w, r)
	name := pat.Param(r, "name")
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
	u := s.charmUserFromRequest(w, r)
	name := pat.Param(r, "name")
	seq, err := s.db.NextSeq(u, name)
	if err != nil {
		log.Printf("cannot get next seq: %s", err)
		s.renderError(w)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(&charm.SeqMsg{Seq: seq})
}

func (s *HTTPServer) handlePostFile(w http.ResponseWriter, r *http.Request) {
	u := s.charmUserFromRequest(w, r)
	path := pattern.Path(r.Context())
	f, _, err := r.FormFile("data")
	if err != nil {
		log.Printf("cannot parse form data: %s", err)
		s.renderError(w)
		return
	}
	defer f.Close()
	err = s.cfg.FileStore.Put(u.CharmID, path, f)
	if err != nil {
		log.Printf("cannot post file: %s", err)
		s.renderError(w)
		return
	}
}

func (s *HTTPServer) handleGetFile(w http.ResponseWriter, r *http.Request) {
	u := s.charmUserFromRequest(w, r)
	path := pattern.Path(r.Context())
	f, err := s.cfg.FileStore.Get(u.CharmID, path)
	if err == ErrFileNotFound {
		s.renderCustomError(w, "file not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("cannot get file: %s", err)
		s.renderError(w)
		return
	}
	defer f.Close()
	switch f.(type) {
	case *dirBuffer:
		w.Header().Set("Content-Type", "application/json")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	_, err = io.Copy(w, f)
	if err != nil {
		log.Printf("cannot copy file: %s", err)
		s.renderError(w)
		return
	}
}

func (s *HTTPServer) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	u := s.charmUserFromRequest(w, r)
	path := pattern.Path(r.Context())
	err := s.cfg.FileStore.Delete(u.CharmID, path)
	if err != nil {
		log.Printf("cannot delete file: %s", err)
		s.renderError(w)
		return
	}
}

func (s *HTTPServer) charmUserFromRequest(w http.ResponseWriter, r *http.Request) *charm.User {
	u := r.Context().Value(ctxUserKey)
	if u == nil {
		log.Printf("could not assign user to request context")
		s.renderError(w)
	}
	return u.(*charm.User)
}
