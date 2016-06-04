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
	client, err := New(nil)
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client, t *testing.T) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client, t)
	// Create server.
	addr := "sleuth-test"
	server, err := New(&Config{
		Handler: http.FileServer(http.Dir(".")), Service: addr})
	if err != nil {
		t.Errorf("server instantiation failed: %s", err.Error())
		return
	}
	defer func(server *Client, t *testing.T) {
		if err := server.Close(); err != nil {
			t.Errorf("server close failed: %s", err.Error())
		}
	}(server, t)
	// Wait until the server has been added to the client pool.
	client.WaitFor(addr)
	if client.block(addr) {
		t.Errorf("call to block should have returned immediately")
	}
	request, err := http.NewRequest("GET", scheme+"://"+addr+"/README.md", nil)
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
	client, err := New(nil)
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client, t *testing.T) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client, t)
	// Create server.
	server, err := New(&Config{
		Handler: http.FileServer(http.Dir(".")),
		Service: "sleuth-test-server-one"})
	if err != nil {
		t.Errorf("server instantiation failed: %s", err.Error())
		return
	}
	defer func(server *Client, t *testing.T) {
		if err := server.Close(); err != nil {
			t.Errorf("server close failed: %s", err.Error())
		}
	}(server, t)
	// Wait until the server has been added to the client pool.
	client.WaitFor("sleuth-test-server-one")
}

func TestBadConfig(t *testing.T) {
	// Create server.
	_, err := New(&Config{
		Handler: http.FileServer(http.Dir("."))})
	if err == nil {
		t.Errorf("server instantiation should fail with bad config")
		return
	}
}

func TestTimeout(t *testing.T) {
	// Create server.
	addr := "sleuth-test-server-three"
	server, err := New(&Config{Handler: new(silentHandler), Service: addr})
	if err != nil {
		t.Errorf("server instantiation failed: %s", err.Error())
		return
	}
	defer func(server *Client, t *testing.T) {
		if err := server.Close(); err != nil {
			t.Errorf("server close failed: %s", err.Error())
		}
	}(server, t)
	// Create client.
	client, err := New(nil)
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client, t *testing.T) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client, t)
	// Wait long enough that the server should be ready (but not guaranteed).
	// If the server is ready, then WaitFor will not block.
	<-time.After(2000 * time.Millisecond)
	// Wait until the server has been added to the client pool.
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
	client, err := New(nil)
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client, t *testing.T) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client, t)
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
	client, err := New(nil)
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client, t *testing.T) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client, t)
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
	client, err := New(&Config{LogLevel: "foo"})
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client, t *testing.T) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client, t)
}

func TestBadServerInstantiation(t *testing.T) {
	Debug = true
	defer func() { Debug = false }()
	_, err := New(&Config{
		Handler: new(silentHandler),
		Service: "sleuth-test-server-four",
		Port:    1})
	if err == nil {
		t.Errorf("server instantiation with restricted port should fail")
		return
	}
}

func TestSimultaneousPosts(t *testing.T) {
	// Create server.
	addr := "sleuth-test-server-five"
	server, err := New(&Config{Handler: new(repeatHandler), Service: addr})
	if err != nil {
		t.Errorf("server instantiation failed: %s", err.Error())
		return
	}
	defer func(server *Client, t *testing.T) {
		if err := server.Close(); err != nil {
			t.Errorf("server close failed: %s", err.Error())
		}
	}(server, t)
	// Create client.
	client, err := New(nil)
	// Set timeout to 10 seconds to accommodate slow test spin-up.
	client.Timeout = time.Second * 10
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client, t *testing.T) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client, t)
	// Wait until the server has been added to the client pool.
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
	for range done {
		if total++; total == requests {
			return
		}
	}
}

func TestBadInterfaceInConfig(t *testing.T) {
	// Create client.
	if _, err := New(&Config{Interface: "foo"}); err == nil {
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
	client, err := New(nil)
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client, t *testing.T) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client, t)
	name := "foo"
	node := "bar"
	service := ""
	version := ""
	err = client.add(group, name, node, service, version)
	if err == nil {
		t.Errorf("expected bad group member addition to fail")
	}
}

func TestDispatchEmptyPayload(t *testing.T) {
	// Create client.
	client, err := New(nil)
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client, t *testing.T) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client, t)
	payload := []byte{}
	if err := client.dispatch(payload); err == nil {
		t.Errorf("expected empty payload to fail dispatch")
	}
}

func TestDispatchBadAction(t *testing.T) {
	// Create client.
	client, err := New(nil)
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client, t *testing.T) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client, t)
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

func TestReceiveBadUnzipPayload(t *testing.T) {
	// Create client.
	client, err := New(nil)
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client, t *testing.T) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client, t)
	payload := []byte("")
	if err := client.receive(payload); err == nil {
		t.Errorf("expected bad payload to fail to unzip")
	}
}

func TestReceiveBadHandlePayload(t *testing.T) {
	// Create client.
	client, err := New(nil)
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client, t *testing.T) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client, t)
	res := new(response)
	res.Handle = "foo"
	payload := marshalResponse(res)[len(group)+len(recv):]
	if err := client.receive(payload); err == nil {
		t.Errorf("expected bad payload to fail on handle check")
	}
}

func TestReplyBadUnmarshalPayload(t *testing.T) {
	// Create client.
	client, err := New(nil)
	if err != nil {
		t.Errorf("client instantiation failed: %s", err.Error())
		return
	}
	defer func(client *Client, t *testing.T) {
		if err := client.Close(); err != nil {
			t.Errorf("client close failed: %s", err.Error())
		}
	}(client, t)
	payload := []byte("")
	if err := client.reply(payload); err == nil {
		t.Errorf("expected bad payload to fail on unmarshal")
	}
}

func TestError(t *testing.T) {
	code := 1
	message := "test"
	want := "sleuth: test [1]"
	err := newError(code, message)
	if err.Error() != want {
		t.Errorf("expected error to be formatted as: %s", want)
	}
}
