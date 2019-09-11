package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
)

type Server struct {
	imageIdx uint64
	images   []string
	delay    time.Duration
}

func MakeServer() *Server {
	return &Server{}
}

func (s *Server) WithImages(images []string) *Server {
	s.images = images
	return s
}

func (s *Server) WithDelay(delay string) *Server {
	var err error
	s.delay, err = time.ParseDuration(delay)
	if err != nil {
		panic("Invalid duration")
	}
	return s
}

func (s *Server) MakeRouter() http.Handler {
	router := mux.NewRouter()

	router.HandleFunc("/random/", s.handleRandom())

	router.HandleFunc("/ordered/", s.handleOrdered())

	return router
}

func (s *Server) handleRandom() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idx := rand.Intn(len(s.images))
		log.Printf("Serving %v", s.images[idx])
		img, err := os.Open(s.images[idx])
		if err != nil {
			http.Error(w, "Failed to open image", http.StatusInternalServerError)
		}
		defer img.Close()
		w.Header().Set("Content-Type", "image/jpeg")
		n, err := io.Copy(w, img)
		if err != nil {
			log.Printf("Error serving %v: %v", s.images[idx], err)
		}
		w.Header().Set("Content-length", strconv.FormatInt(n, 10))
	}
}

func (s *Server) handleOrdered() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idx := atomic.AddUint64(&s.imageIdx, 1)
		idx = idx % uint64(len(s.images))
		log.Printf("Serving %v", s.images[idx])
		time.Sleep(time.Millisecond)
		img, err := os.Open(s.images[idx])
		if err != nil {
			http.Error(w, "Failed to open image", http.StatusInternalServerError)
		}
		defer img.Close()
		w.Header().Set("Content-Type", "image/jpeg")
		n, err := io.Copy(w, img)
		if err != nil {
			log.Printf("Error serving %v: %v", s.images[idx], err)
		}
		w.Header().Set("Content-length", strconv.FormatInt(n, 10))
	}
}

func main() {
	var port *uint = flag.Uint("port", 80, "port on which to expose the API")
	var imagesDir *string = flag.String("path", "", "directory to serve images from")
	var delay *string = flag.String("delay", "100ms", "time to wait before responding")
	flag.Parse()

	if *imagesDir == "" {
		log.Fatal("missing -path argument")
	}

	images, err := filepath.Glob(filepath.Clean(*imagesDir) + "/*.jpg")
	if err != nil {
		log.Fatal(err)
	}

	imgServer := MakeServer().WithImages(images).WithDelay(*delay)
	router := imgServer.MakeRouter()

	address := fmt.Sprintf(":%d", *port)

	httpServer := &http.Server{
		Addr:         address,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      router,
	}

	log.Printf("Serving at %v\n", address)
	log.Fatal(httpServer.ListenAndServe())
}
