# sleuth <img src="https://cdn.rawgit.com/ursiform/sleuth/662e1c96d211b0d27c30fbfa043edc4b3bd6c35d/logo.svg" height="50" valign="middle">
[![API documentation](https://godoc.org/github.com/ursiform/sleuth?status.svg)](https://godoc.org/github.com/ursiform/sleuth) [![Coverage Status](https://coveralls.io/repos/github/ursiform/sleuth/badge.svg)](https://coveralls.io/github/ursiform/sleuth?branch=master)


`sleuth` is a Go library that provides master-less peer-to-peer autodiscovery and RPC
between HTTP services that reside on the same network. It works with minimal
configuration and provides a mechanism to join a local network both as a
client that offers no services and as any service that speaks HTTP. Its
primary use case is for microservices on the same network that make calls to
one another.

For a full introduction and tutorial, check out: [Service autodiscovery in Go with sleuth](http://darian.af/post/master-less-peer-to-peer-micro-service-autodiscovery-in-golang-with-sleuth/)

## Installation
`sleuth` is dependent on [`libzmq`](https://github.com/zeromq/libzmq), which can be installed either from source or from binaries. For more information, please refer to [ØMQ: "Get the Software"](http://zeromq.org/intro:get-the-software) or the [`libzmq` repository](https://github.com/zeromq/libzmq).

Another option is to use a [Docker container that comes with Go and ZeroMQ](https://hub.docker.com/r/rxwen/golang-zeromq/).

Once `libzmq` is available on a system, `sleuth` can be installed like any other Go library:

```
go get -u github.com/ursiform/sleuth
```
## API
The [`sleuth` API documentation is available on GoDoc](https://godoc.org/github.com/ursiform/sleuth) or you can simply run:

```
godoc github.com/ursiform/sleuth
```


## Examples
**Example (1):** The `echo-service` is a toy service that merely echoes back anything in an HTTP request body. It has made itself available on a `sleuth` network:
```go
package main

import (
  "io/ioutil"
  "net/http"

  "github.com/ursiform/sleuth"
)

type echoHandler struct{}

func (h *echoHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
  body, _ := ioutil.ReadAll(req.Body)
  res.Write(body)
}

func main() {
  handler := new(echoHandler)
  // In the real world, the Interface field of the sleuth.Config object
  // should be set so that all services are on the same subnet.
  config := &sleuth.Config{Handler: handler, Service: "echo-service"}
  server, err := sleuth.New(config)
  if err != nil {
    panic(err.Error())
  }
  defer server.Close()
  http.ListenAndServe(":9873", handler)
}
```

And here is a trivial client that waits until it has connected to the network and found the `echo-service` to make a request before it exits. Note that the `*sleuth.Client` works as a drop-in replacement for an `*http.Client` when making requests using the `Do` method:

```go
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ursiform/sleuth"
)

func main() {
	service := "echo-service"
	// In the real world, the Interface field of the sleuth.Config object
	// should be set so that all services are on the same subnet.
	config := &sleuth.Config{LogLevel: "silent"}
	client, err := sleuth.New(config)
	if err != nil {
		panic(err.Error())
	}
	defer client.Close()
	client.WaitFor(service)
	input := "This is the value I am inputting."
	body := bytes.NewBuffer([]byte(input))
	request, _ := http.NewRequest("POST", "sleuth://"+service+"/", body)
	response, err := client.Do(request)
	if err != nil {
		panic(err.Error())
	}
	output, _ := ioutil.ReadAll(response.Body)
	if string(output) == input {
		fmt.Println("It works.")
	} else {
		fmt.Println("It doesn't work.")
	}
}
```

---

**Example (2):**  [`sleuth-example` is a fuller example of two services on a `sleuth` network](https://github.com/afshin/sleuth-example/) that need to communicate with each other.

A complete tutorial based on that example can be found here: [Service autodiscovery in Go with sleuth](http://darian.af/post/master-less-peer-to-peer-micro-service-autodiscovery-in-golang-with-sleuth/).

## Test
    go test -cover github.com/ursiform/sleuth

## Q & A

**Q**: What is the messaging protocol `sleuth` uses?

**A**: Under the hood, `sleuth` marshals HTTP requests and responses into plain JSON objects and then compresses them via `gzip`. Instead of adding another dependency on something like Protocol Buffers, `sleuth` depends on the fact that most API responses between microservices will be fairly small and it leaves the door open to ports in a wide variety of languages and environments. One hard dependency seemed quite enough.

---

**Q**: What if I have multiple instances of the same service?

**A**: Great! `sleuth` will automatically round-robin the requests each client makes to all services that share the same name.

---

**Q**: What happens if a service goes offline?

**A**: Whenever possible, a service should call its client's `Close()` method before exiting to notify the network of its departure. But even if a service fails to do that, the `sleuth` network's underlying `Gyre` network will detect within about one second that a peer has disappeared. All requests to that service will be routed to other peers offering the same service. If no peers exist for that service, then calls to `Do()` will return an unknown service error (code `919`), which means that if you're already handling errors when making requests, you're covered.

---

**Q**: It doesn't work.

**A**: That's not a question. But have you checked to make sure your firewall allows `UDP` traffic on port `5670`?

---

**Q**: It still doesn't work.

**A**: That's still not a question. But have you set the `Interface` field of your `sleuth.Config` object? The services you want to connect need to be on the same network and if you leave that field blank, the underlying `Gyre` network may not reside where you think it does. If you run `ifconfig` you'll get a list of available interfaces on your system.

---

**Q**: Why is it called `sleuth`?

**A**: Because "sleuth" is the collective noun for a group of bears: the original reason for writing this library was to connect a group of [bear](https://github.com/ursiform/bear)/[forest](https://github.com/ursiform/forest) services. Also because a sleuth searches for things and discovers them. Hence the logo:

<p align="center">
    <img src="https://cdn.rawgit.com/ursiform/sleuth/662e1c96d211b0d27c30fbfa043edc4b3bd6c35d/logo.svg">
</p>

## License
`sleuth` is licensed under the [MIT License](LICENSE).

The underlying libraries that `sleuth` relies on, [`Gyre`](https://github.com/zeromq/gyre) and [`libzmq`](https://github.com/zeromq/libzmq), are licensed under the [LGPL](http://www.gnu.org/licenses/lgpl-3.0.en.html). In effect, users who do not plan on modifying `Gyre` or `libzmq` can release their own applications under any license they see fit.

## Resources

* [API documentation](https://godoc.org/github.com/ursiform/sleuth)
* [Service autodiscovery in Go with sleuth](http://darian.af/post/master-less-peer-to-peer-micro-service-autodiscovery-in-golang-with-sleuth/) (tutorial)
