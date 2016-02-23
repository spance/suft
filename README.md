# What is the name then?

[![Join the chat at https://gitter.im/spance/suft](https://badges.gitter.im/spance/suft.svg)](https://gitter.im/spance/suft?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

Small-scale UDP Fast Transmission Protocol, SUFT ?

The SUFT is a application layer transmission protocol based on UDP and implemented in Golang. It provides some of the same service features: stream-oriented like TCP, low latency and esures reliable, ordered transport of stream with plain congestion control.

The protocol is designed for maximizing throughput and minimizing effect of losing packets on throughput, and it oriented in small and medium-scale communication scene.

# Goals & Features

- Transmitting model has predictable performance
- Fast retransmission could better deal with lossy link
- Minimum retransmission without wasting traffic
- No resource consumption while the connection is idle

# Protocol APIs

SUFT implements the [Golang: net.Conn](https://golang.org/pkg/net/#Conn) interface completely.

```
e, err := suft.NewEndpoint(laddr string)
conn := e.Dial(raddr string) // for client
conn := e.Listen() // for server
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

# Known Issues

1. Numbers

   seq, ack... use uint32, that means the connection cannot transmit more than 4GB data.

2. Detecting channel capacity

   to do or NOT ?

3. Test

   need a lot of testing under real-world scene.

# License

GPLv2

# Tool Usage

main package include a tool for testing, similar to netcat (nc).

build with `go build -o suft-nc`

```
./suft-nc [-l addr:port] [-r addr:port] [-s] [-b 10] [-fr] < [send_file] > [recv_file]

-l:  local bind address, e.g. localhost:9090 or :8080
-r:  remote address (for client), e.g. 8.8.8.8:9090 or examples.com:8080
-b:  max bandwidth of sending in mbps
-s:  for server
-fr: enable fast retransmission (useful for lossy link)
-sr: don't shrink window when a lot of packets were lost
-debug: 0,1,2 for debugging
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

- the target will be connected can't be located behind NAT (or should use port mapping)
- use improper bandwidth(-b) may waste huge bandwidth and may be suspected of being used for the purpose of attack.
