package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"github.com/umeshlumbhani/go-wifi-connect/internal/interfaces"
	"github.com/umeshlumbhani/go-wifi-connect/internal/models"
)

// HTTPServer represents a module that provides an https protocol
// to link between our application and the Bridge
type HTTPServer struct {
	Log             *logrus.Logger
	Cfg             models.ConfigHandler
	NetworkManager  interfaces.Network
	Server          *http.Server
	isServerStarted bool
}

// spaHandler implements the http.Handler interface, so we can use it
// to respond to HTTP requests. The path to the static directory and
// path to the index file within that static directory are used to
// serve the SPA in the given static directory.
type spaHandler struct {
	staticPath string
	indexPath  string
}

// ConnectRequest request object for connect
type ConnectRequest struct {
	Identity   string `json:"identity"`
	Passphrase string `json:"passphrase"`
	SSID       string `json:"ssid"`
}

// NewHTTPServer creates an HTTP health checker
func NewHTTPServer(l *logrus.Logger, nw interfaces.Network, cfg models.ConfigHandler) *HTTPServer {
	return &HTTPServer{
		Log:             l,
		Cfg:             cfg,
		NetworkManager:  nw,
		isServerStarted: false,
	}
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path, err := filepath.Abs(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	path = filepath.Join(h.staticPath, path)
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		http.ServeFile(w, r, filepath.Join(h.staticPath, h.indexPath))
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.FileServer(http.Dir(h.staticPath)).ServeHTTP(w, r)
}

// Handler returns Router
func (h *HTTPServer) Handler() http.Handler {
	router := mux.NewRouter()
	cfg := h.Cfg.Fetch()
	router.HandleFunc("/networks", h.GetNetworks).Methods("GET")
	router.HandleFunc("/connect", h.Connect).Methods("POST")

	spa := spaHandler{staticPath: cfg.UIDirectory, indexPath: "index.html"}
	router.PathPrefix("/").Handler(spa)
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedHeaders: []string{"Content-Type", "Authorization", "Content-Length", "X-Requested-With", "Accept", "Origin"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
	})
	handler := c.Handler(router)
	return handler
}

//StartHTTPServer retuns http.Server
func (h *HTTPServer) StartHTTPServer() {
	h.Log.Info("Start HTTP Server")
	s := &http.Server{
		Addr:         fmt.Sprintf(":%s", h.Cfg.Fetch().Port),
		Handler:      h.Handler(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  5 * time.Second,
	}

	go func() {
		h.Log.Info("HTTP Server starting.....")
		err := s.ListenAndServe()
		if err != nil {
			// log error
			h.Log.Error(fmt.Sprintf("startHTTPServer - Error while serving health check : %v", err))
			h.Log.Error("HTTPServer Closed")
		}
	}()

	h.Server = s
}

// CloseHTTPServer used to close HTTP server
func (h *HTTPServer) CloseHTTPServer() {
	if h.isServerStarted && h.Server != nil {
		h.Server.Close()
		h.isServerStarted = false
		h.Server = nil
	}
}

// GetNetworks method used to retrieve list of networks
func (h *HTTPServer) GetNetworks(w http.ResponseWriter, r *http.Request) {
	h.Log.Info("'GetNetworks' called via http request")
	ap, err := h.NetworkManager.GetAccessPoint()
	if err != nil {
		respondWithError(w, 500, "Internal Error")
		return
	}
	respondWithJSON(w, http.StatusOK, ap)
}

// Connect method used to connect
func (h *HTTPServer) Connect(w http.ResponseWriter, r *http.Request) {
	h.Log.Info("'Connect' called via http request")
	decoder := json.NewDecoder(r.Body)
	var req ConnectRequest
	err := decoder.Decode(&req)
	if err != nil {
		h.Log.Error(fmt.Sprintf("Connect - found error on extract request body: %s", err.Error()))
		respondWithError(w, 400, "Bad Request")
		return
	}
	ok := h.NetworkManager.Connect(req.SSID, req.Passphrase, req.Identity)
	if !ok {
		respondWithJSON(w, 500, "internal error")
		return
	}
	respondWithJSON(w, 200, nil)
	os.Exit(0)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		response = []byte{}
		code = http.StatusInternalServerError
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
