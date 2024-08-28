// Copyright (C) 2024 The Dagu Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package sock

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/dagu-org/dagu/internal/logger"
)

var ErrServerRequestedShutdown = errors.New(
	"socket frontend is requested to shutdown",
)

// Server is a unix socket frontend that passes http requests to HandlerFunc.
type Server struct {
	addr        string
	handlerFunc HTTPHandlerFunc
	listener    net.Listener
	quit        atomic.Bool
	mu          sync.Mutex
	logger      logger.Logger
}

// HTTPHandlerFunc is a function that handles HTTP requests.
type HTTPHandlerFunc func(w http.ResponseWriter, r *http.Request)

// NewServer creates a new unix socket frontend.
func NewServer(
	addr string,
	handlerFunc HTTPHandlerFunc,
	lg logger.Logger,
) (*Server, error) {
	return &Server{
		addr:        addr,
		handlerFunc: handlerFunc,
		logger:      lg,
	}, nil
}

// Serve starts listening and serving requests.
func (srv *Server) Serve(listen chan error) error {
	_ = os.Remove(srv.addr)
	var err error
	srv.listener, err = net.Listen("unix", srv.addr)
	if err != nil {
		if listen != nil {
			listen <- err
		}
		return err
	}
	if listen != nil {
		listen <- err
	}
	srv.logger.Debug("Unix socket is listening", "addr", srv.addr)

	defer func() {
		_ = srv.Shutdown()
		_ = os.Remove(srv.addr)
	}()
	for {
		conn, err := srv.listener.Accept()
		if srv.quit.Load() {
			return ErrServerRequestedShutdown
		}
		if err == nil {
			go func() {
				request, err := http.ReadRequest(bufio.NewReader(conn))
				if err != nil {
					srv.logger.Error("read request", "error", err)
				} else {
					srv.handlerFunc(newHTTPResponseWriter(&conn), request)
				}
				_ = conn.Close()
			}()
		}
	}
}

// Shutdown stops the frontend.
func (srv *Server) Shutdown() error {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if !srv.quit.Load() {
		srv.quit.Store(true)
		if srv.listener != nil {
			err := srv.listener.Close()
			if err != nil && !errors.Is(err, os.ErrClosed) {
				srv.logger.Error("close listener", "error", err)
			}
			return err
		}
	}
	return nil
}

var _ http.ResponseWriter = (*httpResponseWriter)(nil)

type httpResponseWriter struct {
	conn       *net.Conn
	header     http.Header
	statusCode int
}

func newHTTPResponseWriter(conn *net.Conn) http.ResponseWriter {
	return &httpResponseWriter{
		conn:       conn,
		header:     make(http.Header),
		statusCode: http.StatusOK,
	}
}

func (w *httpResponseWriter) Write(data []byte) (int, error) {
	response := http.Response{
		StatusCode: w.statusCode,
		ProtoMajor: 1,
		ProtoMinor: 0,
		Body:       io.NopCloser(strings.NewReader(string(data))),
		Header:     w.header,
	}
	_ = response.Write(*w.conn)
	return 0, nil
}

func (w *httpResponseWriter) Header() http.Header {
	return w.header
}

func (w *httpResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}
