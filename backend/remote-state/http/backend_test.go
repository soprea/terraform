package http

import (
	"github.com/hashicorp/terraform/backend"

	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)
func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestBackendConfig(t *testing.T) {
	handler := new(testHTTPHandler)
	ts := httptest.NewServer(http.HandlerFunc(handler.Handle))
	defer ts.Close()
	url, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	urls := fmt.Sprintf("%s", url)
	// Build config
	config := map[string]interface{}{
		"address": urls,
	}
	b := backend.TestBackendConfig(t, New(), config).(*Backend)
	if b.address != urls {
		t.Fatal("Incorrect url was provided.")
	}
}
/*
func TestBackendStates(t *testing.T) {
	handler := new(testHTTPHandler)
	ts := httptest.NewServer(http.HandlerFunc(handler.Handle))
	defer ts.Close()
	url, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	urls := fmt.Sprintf("%s", url)
	// Build config
	config := map[string]interface{}{
		"address": urls,
	}
	b := backend.TestBackendConfig(t, New(), config).(*Backend)
	backend.TestBackendStates(t, b)
}
*/
func TestBackendLocked(t *testing.T) {
	handler := new(testHTTPHandler)
	ts := httptest.NewServer(http.HandlerFunc(handler.Handle))
	defer ts.Close()
	url, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	urls := fmt.Sprintf("%s", url)
	// Build config
	config := map[string]interface{}{
		"address": urls,
	}
	b1 := backend.TestBackendConfig(t, New(), config).(*Backend)
	b2 := backend.TestBackendConfig(t, New(), config).(*Backend)
	backend.TestBackendStateLocks(t, b1, b2)
	//backend.TestBackendStateForceUnlock(t, b1, b2)
}

type testHTTPHandler struct {
	Data   map[string][]byte
	Locked bool
}

func (h *testHTTPHandler) Handle(w http.ResponseWriter, r *http.Request) {
	log.Printf("[DEBUG] TEST Stefan request Method: [%+v]", r.Method)
	log.Printf("[DEBUG] TEST Stefan request URL: [%+v]", r.URL)
	log.Printf("[DEBUG] TEST Stefan h.Data: [%+v]", h.Data)
	if h.Data == nil {
		h.Data = make(map[string][]byte)
		//h.Data["/"] = []byte("default")
	}
	switch r.Method {
	case http.MethodGet:
		switch r.URL.Path {
		case "/foo.tfstate":
			w.Write(h.Data["/foo.tfstate"])
		case "/bar.tfstate":
			w.Write(h.Data["/bar.tfstate"])
		case "/default.tfstate":
			w.Write(h.Data["/default.tfstate"])
		case "/":
			var keys []string
			for key, _ := range h.Data {
        // we already return default state from the backend_state
				keys = append(keys, key)
			}
			all := fmt.Sprint(strings.Join(keys, ","))
			alls := []byte(all)
			h.Data["/"] = alls
			log.Printf("[DEBUG] TEST Stefan keys: [%+v]", all)
			log.Printf("[DEBUG] TEST Stefan / key value is: [%+v]", h.Data["/"])
			w.Write(h.Data["/"])
		}

	case "PUT":
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, r.Body); err != nil {
			w.WriteHeader(500)
		}
		w.WriteHeader(201)

	case "POST":
		switch r.URL.Path {
		case "/foo.tfstate":
			buf := new(bytes.Buffer)
			if _, err := io.Copy(buf, r.Body); err != nil {
				w.WriteHeader(500)
			}
			h.Data["/foo.tfstate"] = buf.Bytes()
			log.Printf("[DEBUG] TEST Stefan POST /foo.tfstate: [%+v]", h.Data["/foo.tfstate"])

		case "/bar.tfstate":
			buf := new(bytes.Buffer)
			if _, err := io.Copy(buf, r.Body); err != nil {
				w.WriteHeader(500)
			}
			h.Data["/bar.tfstate"] = buf.Bytes()
			log.Printf("[DEBUG] TEST Stefan POST /bar.tfstate: [%+v]", h.Data["/bar.tfstate"])

		case "/default.tfstate":
			buf := new(bytes.Buffer)
			if _, err := io.Copy(buf, r.Body); err != nil {
				w.WriteHeader(500)
			}
			h.Data["/default.tfstate"] = buf.Bytes()
			log.Printf("[DEBUG] TEST Stefan POST /default.tfstate: [%+v]", h.Data["/default.tfstate"])

		case "/":
			buf := new(bytes.Buffer)
			if _, err := io.Copy(buf, r.Body); err != nil {
				w.WriteHeader(500)
			}
			log.Printf("[DEBUG] TEST Stefan / POST value is: [%+v]", buf.Bytes())

			h.Data["/"] = buf.Bytes()

		}

	case "LOCK":
		log.Printf("[DEBUG] TEST Stefan LOCK request r.URL.Path: [%+v]", r.URL.Path)
		switch r.URL.Path {
		case "/default.tflock":
			log.Printf("[DEBUG] TEST Stefan LOCK /default.tflock: [%s]", h.Data["/default.tflock"])
			log.Printf("[DEBUG] TEST Stefan LOCK h.Locked: [%+v]", h.Locked)
			log.Printf("[DEBUG] TEST Stefan LOCK h.data is: [%+v]", h.Data)
			if h.Locked {
				log.Printf("[DEBUG] TEST Stefan LOCK catched by if")
        w.WriteHeader(http.StatusLocked)
				w.Write([]byte(h.Data["/default.tflock"]))
        //w.WriteHeader(http.StatusLocked)
  			} else {
				log.Printf("[DEBUG] TEST Stefan LOCK catched by else")
				log.Printf("[DEBUG] TEST Stefan LOCK h.data is: [%+v]", h.Data)
				if _, ok := h.Data["/default/tflock"]; ok {
					log.Printf("[DEBUG] TEST Stefan LOCK catched by else if. Would mean already locked.")
					w.WriteHeader(409)
				} else {
					log.Printf("[DEBUG] TEST Stefan LOCK catched by else else")
					log.Printf("[DEBUG] TEST Stefan LOCK catched by else if. Would mean create lock.")
					buf := new(bytes.Buffer)
					if _, err := io.Copy(buf, r.Body); err != nil {
						w.WriteHeader(500)
					}
					h.Data["/default.tflock"] = buf.Bytes()
					log.Printf("[DEBUG] TEST Stefan LOCK h.data is: [%+v]", h.Data)
					h.Locked = true
					log.Printf("[DEBUG] TEST Stefan LOCK h.Locked last: [%+v]", h.Locked)
				}
			}
		}

	case "UNLOCK":
		log.Printf("[DEBUG] TEST Stefan UNLOCK request r.URL.Path: [%+v]", r.URL.Path)
		switch r.URL.Path {
		case "/default.tflock":
			//w.Write(h.Data["/default.tflock"])
			//w.WriteHeader(200)
			h.Locked = false
			log.Printf("[DEBUG] TEST Stefan UNLOCK h.Data before unlock : [%+v]", h.Data)
			delete(h.Data, "/default.tflock")
			log.Printf("[DEBUG] TEST Stefan UNLOCK h.Data after unlock: [%+v]", h.Data)

		}
	case "DELETE":
		switch r.URL.Path {
		// Delete foo.tfstate
		case "/foo.tfstate":
			delete(h.Data, "/foo.tfstate")
			w.WriteHeader(200)
			// Delete bar.tfstate
		case "/bar.tfstate":
			delete(h.Data, "/bar.tfstate")
			w.WriteHeader(200)
		}
	default:
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("Unknown method: %s", r.Method)))
	}

}
