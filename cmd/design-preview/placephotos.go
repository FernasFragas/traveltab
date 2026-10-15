package main

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"weatherservice/internal/adapters/httpserver"
)

// previewServer rewrites only the preview's synthetic Wikimedia markers after the real
// production handlers validate them. Production's photo host policy stays unchanged.
type previewServer struct {
	*httpserver.Server
	localPlacePhotos bool
}

func (s *previewServer) Listen(addr string) error {
	if !s.localPlacePhotos {
		return s.Server.Listen(addr)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	backend := listener.Addr().String()
	if err := listener.Close(); err != nil {
		return err
	}
	go func() { _ = s.Server.Listen(backend) }()
	origin, _ := url.Parse("http://" + backend)
	proxy := httputil.NewSingleHostReverseProxy(origin)
	proxy.ModifyResponse = localPlacePhotoResponse
	mux := http.NewServeMux()
	mux.HandleFunc("/fixture-photos/", fixturePhotoHandler)
	mux.Handle("/", proxy)
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	return server.ListenAndServe()
}

func localPlacePhotoResponse(response *http.Response) error {
	if !strings.HasPrefix(response.Request.URL.Path, "/map/") {
		return nil
	}
	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		return err
	}
	for _, name := range []string{"lisbon-landmark", "lisbon-museum", "lisbon-food", "lisbon-nature"} {
		path := "/fixture-photos/" + name + ".png"
		body = bytes.ReplaceAll(body, []byte("https://upload.wikimedia.org"+path), []byte(path))
	}
	response.Body = io.NopCloser(bytes.NewReader(body))
	response.ContentLength = int64(len(body))
	response.Header.Set("Content-Length", strconv.Itoa(len(body)))
	return nil
}
