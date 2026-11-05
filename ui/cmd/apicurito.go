/*
 * Copyright (C) 2018 Red Hat, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package main

import (
	"crypto/sha1"
	"embed"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

// The Angular application build output (see ../angular.json) is embedded
// into the binary at compile time.
//
//go:embed all:dist
var embedded embed.FS

// server implements fs.FS by first looking in an optional on-disk override
// directory (the --path flag) and then falling back to the embedded assets.
// It replaces the retired go-bindata/go-bindata-assetfs machinery.
type server struct {
	disk    fs.FS // optional on-disk override, nil when not configured
	content fs.FS // embedded "dist" content
}

func (s *server) Open(name string) (fs.File, error) {
	if s.disk != nil {
		if f, err := s.disk.Open(name); err == nil {
			return f, nil
		}
	}
	return s.content.Open(name)
}

// canServe reports whether name resolves to something the static file
// handler can serve (a file, or a directory containing an index.html).
func (s *server) canServe(name string) bool {
	if name == "" {
		name = "."
	}
	file, err := s.Open(name)
	if err != nil {
		return false
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return false
	}
	if info.IsDir() {
		return s.canServe(path.Join(name, "index.html"))
	}
	return true
}

// spaFallback rewrites requests that do not resolve to a static asset to
// the root, so the single page app is served instead of a 404.
func (s *server) spaFallback(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asset := strings.TrimPrefix(r.URL.Path, "/")
		if !s.canServe(asset) {
			// the static content handler redirects /index.html to /
			// so rewrite to / to avoid a redirect.
			r.URL.Path = "/"
		}
		next.ServeHTTP(w, r)
	})
}

// withETag adds a content-hash based Etag header and answers
// If-None-Match requests with 304 Not Modified.
func (s *server) withETag(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asset := strings.TrimPrefix(r.URL.Path, "/")
		if asset == "" {
			asset = "index.html"
		}
		if etag, ok := s.etag(asset); ok {
			w.Header().Set("Etag", etag)
			if r.Header.Get("If-None-Match") == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// etag computes "<sha1>-<modtime>" for the given asset, mirroring the
// format used by previous releases of this server.
func (s *server) etag(name string) (string, bool) {
	file, err := s.Open(name)
	if err != nil {
		return "", false
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil || info.IsDir() {
		return "", false
	}

	sum := sha1.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", false
	}
	modTime := "0"
	if !info.ModTime().IsZero() {
		modTime = strconv.FormatInt(info.ModTime().Unix(), 10)
	}
	return fmt.Sprintf("%x-%s", sum.Sum(nil), modTime), true
}

func main() {
	port := flag.Int("port", 3000, "The port the server accepts connections on")
	dirPath := flag.String("path", "", "The directory path to serve files from (embedded content is used for missing files)")
	prefix := flag.String("prefix", "/", "The URL prefix to serve from")
	spa := flag.Bool("spa", false, "Run in Single Page App mode.")
	etags := flag.Bool("etags", false, "Calculate ETag headers to assist caching")
	flag.Parse()

	content, err := fs.Sub(embedded, "dist")
	if err != nil {
		log.Fatal(err)
	}

	s := &server{content: content}
	if *dirPath != "" {
		if info, err := os.Stat(*dirPath); err == nil && info.IsDir() {
			log.Printf("Serving static content from: %s", *dirPath)
			s.disk = os.DirFS(*dirPath)
		}
	}
	log.Println("Serving static content from embedded resources")

	var handler http.Handler = http.FileServer(http.FS(s))
	if *etags {
		handler = s.withETag(handler)
	}
	if *spa {
		handler = s.spaFallback(handler)
	}
	if *prefix != "/" {
		handler = http.StripPrefix(*prefix, handler)
	}

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Listening on %s", addr)
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 30 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
