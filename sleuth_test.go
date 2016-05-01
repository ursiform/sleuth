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
	client.WaitFor("sleuth-test")
	request, err := http.NewRequest("GET", "/"+ConfigFile, nil)
	if err != nil {
		t.Errorf("request instantiation failed: %s", err.Error())
		return
	}
	response, err := client.Do(request, "sleuth-test")
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
	client.WaitFor("sleuth-test-server-three")
	request, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Errorf("request instantiation failed: %s", err.Error())
		return
	}
	_, err = client.Do(request, "sleuth-test-server-three")
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
	request, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Errorf("request instantiation failed: %s", err.Error())
		return
	}
	if _, err := client.Do(request, "sleuth-test-foo-bar"); err == nil {
		t.Errorf("client request should return unknown service error")
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
	client.WaitFor("sleuth-test-server-five")
	requests := 2
	body := "foo bar baz"
	done := make(chan bool)
	total := 0
	for i := 0; i < requests; i++ {
		go func(t *testing.T, client *Client, done chan bool) {
			defer func() { done <- true }()
			buffer := bytes.NewBuffer([]byte(body))
			request, err := http.NewRequest("POST", "/", buffer)
			if err != nil {
				t.Errorf("request instantiation failed: %s", err.Error())
				return
			}
			response, err := client.Do(request, "sleuth-test-server-five")
			if err != nil {
				t.Errorf("client request failed: %s", err.Error())
				return
			}
			output, _ := ioutil.ReadAll(response.Body)
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
