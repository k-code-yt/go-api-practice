package transport

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/k-code-yt/go-api-practice/protocol-playground/invoicer/ports"
	"github.com/k-code-yt/go-api-practice/protocol-playground/shared"
	"github.com/sirupsen/logrus"
)

type HTTPServer struct {
	port string
}

func NewHTTPServer() ports.ServerTransport {
	return &HTTPServer{
		port: shared.HTTPPortInvoice,
	}
}

func writeJSON(rw http.ResponseWriter, status int, v any) error {
	rw.WriteHeader(status)
	rw.Header().Add("Content-Type", "applicaiton/json")
	return json.NewEncoder(rw).Encode(v)
}

func (s *HTTPServer) SaveInvoice(svc ports.Invoicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logrus.WithFields(logrus.Fields{
			"method":  r.Method,
			"service": "Invoicer",
		}).Info("received save request")
		switch r.Method {
		case http.MethodPost:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid payload"})
			}
			defer r.Body.Close()
			var d shared.Distance
			if err = json.Unmarshal(body, &d); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid payload"})
			}
			invoice := svc.SaveInvoice(&d)
			logrus.WithFields(logrus.Fields{
				"invoice": invoice,
			}).Info("saved invoice SUCCESS")
			writeJSON(w, http.StatusOK, invoice)
		default:
			writeJSON(w, http.StatusNotFound, map[string]string{"message": fmt.Sprintf("%s does not exist", r.Method)})
			return
		}

	}
}

func (s *HTTPServer) Listen(svc ports.Invoicer) error {
	http.HandleFunc("/invoice", s.SaveInvoice(svc))
	logrus.Infof("Listening on port %s\n", s.port)
	return http.ListenAndServe(s.port, nil)
}
