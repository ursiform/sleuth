# sleuth <img src="https://cdn.rawgit.com/ursiform/sleuth/662e1c96d211b0d27c30fbfa043edc4b3bd6c35d/logo.svg" height="50" valign="middle"> [![Coverage Status](https://coveralls.io/repos/github/ursiform/sleuth/badge.svg?branch=master)](https://coveralls.io/github/ursiform/sleuth?branch=master) [![API documentation](https://godoc.org/github.com/ursiform/sleuth?status.svg)](https://godoc.org/github.com/ursiform/sleuth)


`sleuth` is a Go library that provides master-less peer-to-peer autodiscovery and RPC
between HTTP services that reside on the same network. It works with minimal
configuration and provides a mechanism to join a local network both as a
client that offers no services and as any service that speaks HTTP. Its
primary use case is for microservices on the same network that make calls to
one another.

## Installation
`sleuth` is dependent on [`libzmq`](https://github.com/zeromq/libzmq), which can be installed either from source or from binaries. For more information, please refer to [ØMQ: "Get the Software"](http://zeromq.org/intro:get-the-software) or the [`libzmq` repository](https://github.com/zeromq/libzmq).

Once `libzmq` is available on a system, `sleuth` can be installed like any other Go library:

```
go get -u github.com/ursiform/sleuth
```

## Test
    go test -cover github.com/ursiform/sleuth

## Q & A

**Q**: What is the messaging protocol `sleuth` uses?

**A**: Under the hood, `sleuth` marshals HTTP requests and responses into plain JSON objects and then compresses them via `gzip`. Instead of adding another dependency on something like Protocol Buffers, `sleuth` depends on the fact that most API responses between microservices will be fairly small and it leaves the door open to ports in a wide variety of languages and environments. One hard dependency seemed quite enough.

**Q**: It doesn't work.

**A**: That's not a question. But have you checked to make sure your firewall allows `UDP` traffic on port `5670`?

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
