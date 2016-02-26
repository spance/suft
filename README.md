# Introduction

[![Join the chat at https://gitter.im/spance/suft](https://badges.gitter.im/spance/suft.svg)](https://gitter.im/spance/suft?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![GoDoc](https://godoc.org/github.com/spance/suft/protocol?status.svg)](https://godoc.org/github.com/spance/suft/protocol)

Small-scale UDP Fast Transmission Protocol, SUFT ?

The SUFT is a application layer transmission protocol based on UDP and implemented in Golang. It provides some of the same service features: stream-oriented like TCP, low latency and esures reliable, ordered transport of stream with plain congestion control.

The protocol is designed for maximizing throughput and minimizing effect of losing packets on throughput, and it just oriented in small and medium-scale communication scene or some TCP incapable situations.

# Goals & Features

- Transmitting model has predictable performance
- Fast retransmission could better deal with lossy link
- Minimum retransmission without wasting traffic
- No resource consumption while the connection is idle

# Protocol APIs

SUFT implements the Golang: [net.Conn](https://golang.org/pkg/net/#Conn) and [net.Listener](https://golang.org/pkg/net/#Listener) interfaces completely.

```go
import "github.com/spance/suft/protocol"

e, err := suft.NewEndpoint(p *suft.Params)
// for server
conn := e.Listen() // or e.Accept()
// for client
conn, err := e.Dial(rAddr string)
```

# Basic Theories

Sent-count(aka, the count of unique data packets) and lost-count

```
    lscnt
-------------- = Lose Rate
 scnt + lscnt

    lscnt
-------------- = Retransmit Rate
    scnt
```

latency, window and traffic speed

```
  1000
--------- * mss * win = Speed
 latency
```

# License

GPLv2

# Tool Usage

main package include a tool for testing, similar to netcat (nc).

build with `go get -u -v github.com/spance/suft/suft-nc`

```
./suft-nc [-l addr:port] [-r addr:port] [-s] [-b 10] [-fr] < [send_file] > [recv_file]

-l:  local bind address, e.g. localhost:9090 or :8080
-r:  remote address (for client), e.g. 8.8.8.8:9090 or examples.com:8080
-b:  max bandwidth of sending in mbps (be careful, see Notes#2)
-s:  for server
-fr: enable fast retransmission (useful for lossy link)
-sr: don't shrink window when a lot of packets were lost
```

examples:

```
// send my_file to remote in 10mbps
remote# ./suft-nc -l :9090 -s > recv_file
local# ./suft-nc -l :1234 -r remote:9090 -b 10 -fr < my_file
```

```
// recv my_file from remote in 20mbps
remote# ./suft-nc -l :9090 -s -b 20 -fr < my_file
local# ./suft-nc -l :1234 -r remote:9090 > recv_file
```

```
// simple chat room
remote# ./suft-nc -l :9090 -s
local# ./suft-nc -l :1234 -r remote:9090
```

Notes:

1. the target will be connected can't be located behind NAT (or should use port mapping)
2. use improper bandwidth(-b) may waste huge bandwidth and may be suspected of being used for the purpose of attack.

# Known Issues

1. Numbers

   seq, ack... use uint32, that means one connection cannot transmit more than 4G packets (bytes ∈ [4GB, 5.6TB]).

2. Detecting channel capacity

   to do or NOT ?

3. Test

   need a lot of testing under real-world scene. Welcome to share your test results.
