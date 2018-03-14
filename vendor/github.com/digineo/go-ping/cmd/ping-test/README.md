# ping-test

This is a sample program to demonstrate the usage of this library.
It is not intended for production use.

## How-to

**Building/installing** does not require any special steps. Either run
`go get` (to install it directly in `$GOPATH/bin/ping-test`)

```
$ go get -u github.com/digineo/go-ping/cmd/ping-test
```

or skip installing it (and build it yourself):

```
$ go get -u -d github.com/digineo/go-ping
$ cd $GOPATH/src/github.com/digineo/go-ping/cmd/ping-test
$ go build    # this creates ./ping-test
```

**Running** `ping-test` requires elevated privileges, since normal users
cannot open ICMP sockets.

To circumvent this, either run the binary as root, e.g. via `sudo`
 *(not recommended!)*

```
$ sudo ./ping-test -4 golang.org
ping golang.org (216.58.211.113) rtt=11.869403ms
$ sudo ./ping-test -6 golang.org
ping golang.org (2a00:1450:400e:809::2011) rtt=11.412907ms
```

Better yet, allow the binary to only open raw sockets (via `capabilities(7)`):

```
$ sudo setcap cap_net_raw+ep ./ping-test
$ ./ping-test -4 golang.org
ping golang.org (216.58.211.113) rtt=11.772573ms
$ ./ping-test -6 golang.org
ping golang.org (2a00:1450:400e:809::2011) rtt=11.31439ms
```

Note, that you'll need to re-apply the `setcap` command everytime the
binary changes (i.e. after `go build`).

Also, since configuring the system capabilities is a Linux feature, you
may need to resort to Docker or VM environments, if you want to try
this binary, but don't trust its source code. Or you like living in the
danger zone and don't mind the occasional system crash introduced by
running code found on the internet with root privileges :-)
