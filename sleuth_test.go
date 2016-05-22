package sleuth

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

func init() {
	// Tests should run using a different group than production.
	group = "SLEUTH-vT"
}

type repeatHandler struct{}

func (h *repeatHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	body, _ := ioutil.ReadAll(req.Body)
	res.Write(body)
}

type silentHandler struct{}

func (*silentHandler) ServeHTTP(http.ResponseWriter, *http.Request) {
	// Bad handler does nothing.
}

func TestRequestResponseCycle(t *testing.T) {
	// Create client.
	client, err := New(nil, "")
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client)
	// Create server.
	server, err := New(http.FileServer(http.Dir(".")), "")
	if err != nil {
		t.Errorf("server instantiation failed: %s", err.Error())
		return
	}
	defer func(server *Client) {
		if err := server.Close(); err != nil {
			t.Errorf("server close failed: %s", err.Error())
		}
	}(server)
	// Wait until the server has been added to the client pool.
	addr := "sleuth-test"
	client.WaitFor(addr)
	request, err := http.NewRequest("GET", scheme+"://"+addr+"/"+ConfigFile, nil)
	if err != nil {
		t.Errorf("request instantiation failed: %s", err.Error())
		return
	}
	response, err := client.Do(request)
	if err != nil {
		t.Errorf("client.Do failed: %s", err.Error())
		return
	}
	if response.StatusCode != http.StatusOK {
		t.Errorf("client.Do expected %d got %d", response.StatusCode, http.StatusOK)
		return
	}
}

func TestLoadConfig(t *testing.T) {
	// Create client.
	client, err := New(nil, "test-data/client-1.json")
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client)
	// Create server.
	handler := http.FileServer(http.Dir("."))
	server, err := New(handler, "test-data/server-1.json")
	if err != nil {
		t.Errorf("server instantiation failed: %s", err.Error())
		return
	}
	defer func(server *Client) {
		if err := server.Close(); err != nil {
			t.Errorf("server close failed: %s", err.Error())
		}
	}(server)
	// Wait until the server has been added to the client pool.
	client.WaitFor("sleuth-test-server-one")
}

func TestBadConfig(t *testing.T) {
	// Create server.
	handler := http.FileServer(http.Dir("."))
	_, err := New(handler, "test-data/server-2.json")
	if err == nil {
		t.Errorf("server instantiation should fail with bad config")
		return
	}
}

func TestTimeout(t *testing.T) {
	// Create server.
	handler := new(silentHandler)
	server, err := New(handler, "test-data/server-3.json")
	if err != nil {
		t.Errorf("server instantiation failed: %s", err.Error())
		return
	}
	defer func(server *Client) {
		if err := server.Close(); err != nil {
			t.Errorf("server close failed: %s", err.Error())
		}
	}(server)
	// Create client.
	client, err := New(nil, "")
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client)
	// Wait long enough that the server should be ready (but not guaranteed).
	// If the server is ready, then WaitFor will not block.
	<-time.After(2000 * time.Millisecond)
	// Wait until the server has been added to the client pool.
	addr := "sleuth-test-server-three"
	client.WaitFor(addr)
	request, err := http.NewRequest("GET", scheme+"://"+addr+"/", nil)
	if err != nil {
		t.Errorf("request instantiation failed: %s", err.Error())
		return
	}
	_, err = client.Do(request)
	if err == nil {
		t.Errorf("client request should have timed out")
		return
	}
}

func TestUnknownServiceRequest(t *testing.T) {
	// Create client.
	client, err := New(nil, "")
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client)
	addr := "sleuth-test-foo-bar"
	request, err := http.NewRequest("GET", scheme+"://"+addr+"/", nil)
	if err != nil {
		t.Errorf("request instantiation failed: %s", err.Error())
		return
	}
	if _, err := client.Do(request); err == nil {
		t.Errorf("client request should return unknown service error")
		return
	}
}

func TestUnknownSchemeRequest(t *testing.T) {
	// Create client.
	client, err := New(nil, "")
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client)
	request, err := http.NewRequest("GET", "foo://bar/", nil)
	if err != nil {
		t.Errorf("request instantiation failed: %s", err.Error())
		return
	}
	if _, err := client.Do(request); err == nil {
		t.Errorf("client request should return unknown scheme error")
		return
	}
}

func TestBadLogLevelInConfig(t *testing.T) {
	// Create client. This test is for coverage as it only sets the log level
	// to "debug" but does not change any functionality.
	client, err := New(nil, "test-data/client-2.json")
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client)
}

func TestBadServerInstantiation(t *testing.T) {
	Debug = true
	defer func() { Debug = false }()
	handler := new(silentHandler)
	if _, err := New(handler, "test-data/server-4.json"); err == nil {
		t.Errorf("server instantiation with restricted port should fail")
		return
	}
}

func TestSimultaneousPosts(t *testing.T) {
	// Create server.
	handler := new(repeatHandler)
	server, err := New(handler, "test-data/server-5.json")
	if err != nil {
		t.Errorf("server instantiation failed: %s", err.Error())
		return
	}
	defer func(server *Client) {
		if err := server.Close(); err != nil {
			t.Errorf("server close failed: %s", err.Error())
		}
	}(server)
	// Create client.
	client, err := New(nil, "test-data/client-3.json")
	// Set timeout to 1.5 seconds to accommodate slow test spin-up.
	client.Timeout = time.Millisecond * 1500
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client)
	// Wait until the server has been added to the client pool.
	addr := "sleuth-test-server-five"
	client.WaitFor(addr)
	requests := 2
	body := "foo bar baz"
	done := make(chan struct{})
	total := 0
	for i := 0; i < requests; i++ {
		go func(t *testing.T, client *Client, done chan struct{}) {
			defer func() { done <- struct{}{} }()
			buffer := bytes.NewBuffer([]byte(body))
			request, err := http.NewRequest("POST", scheme+"://"+addr+"/", buffer)
			if err != nil {
				t.Errorf("request instantiation failed: %s", err.Error())
				return
			}
			response, err := client.Do(request)
			if err != nil {
				t.Errorf("client request failed: %s", err.Error())
				return
			}
			output, _ := ioutil.ReadAll(response.Body)
			// It's not really necessary to close a sleuth response.
			response.Body.Close()
			if string(output) != body {
				t.Errorf("expected %s to equal %s", string(output), body)
				return
			}
		}(t, client, done)
	}
	for _ = range done {
		if total++; total == requests {
			return
		}
	}
}

func TestBadInterfaceInConfig(t *testing.T) {
	// Create client.
	if _, err := New(nil, "test-data/client-4.json"); err == nil {
		t.Errorf("client instantiation should have failed")
		return
	}
}

func TestBadResponsePayload(t *testing.T) {
	payload := []byte("{bad json}")
	if _, _, err := unmarshalResponse(payload); err == nil {
		t.Errorf("expected bad response to be unparseable")
	}
}

func TestBadRequestPayload(t *testing.T) {
	payload := []byte("{bad json}")
	if _, _, err := unmarshalRequest(payload); err == nil {
		t.Errorf("expected bad request to be unparseable")
	}
}

func TestAddBadGroupMember(t *testing.T) {
	// Create client.
	client, err := New(nil, "")
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client)
	name := "foo"
	node := "bar"
	service := ""
	version := ""
	if err := client.add(group, name, node, service, version); err == nil {
		t.Errorf("expected bad group member addition to fail")
	}
}

func TestDispatchEmptyPayload(t *testing.T) {
	// Create client.
	client, err := New(nil, "")
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client)
	payload := []byte{}
	if err := client.dispatch(payload); err == nil {
		t.Errorf("expected empty payload to fail dispatch")
	}
}

func TestDispatchBadDispatchAction(t *testing.T) {
	// Create client.
	client, err := New(nil, "")
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client)
	payload := []byte(group + "boom")
	if err := client.dispatch(payload); err == nil {
		t.Errorf("expected bad dispatch action to fail")
	}
}

func TestAddDuplicateServiceWorker(t *testing.T) {
	w := newWorkers()
	p := new(peer)
	p.Name = "test"
	p.Node = "test-node"
	p.Service = "test-service"
	length := w.add(p)
	if length != w.add(p) {
		t.Error("expected duplicate addition to be ignored")
	}
}

func TestRemoveNonExistentServiceWorker(t *testing.T) {
	w := newWorkers()
	_, p := w.remove("test")
	if p != nil {
		t.Error("expected nonexistent service worker removal to return nil")
	}
}

func TestNextNonExistentServiceWorker(t *testing.T) {
	w := newWorkers()
	p := w.next()
	if p != nil {
		t.Error("expected nonexistent service worker to be nil")
	}
}
