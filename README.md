# What is the name then?

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
./suft-nc [-l :port] [-r target:port] [-b 10] [-fr] < [send_file] > [recv_file]

-l:  local bind address
-r:  remote address
-b:  max bandwidth of sending in mbps
-fr: enable fast retransmission (useful for a link with losing packets)
-sr: don't shrink window when a lot of packets were lost
-debug: 0,1,2 for debugging
```

Notes:

- the target will be connected can't be located behind NAT (or should use port mapping)
- use improper bandwidth(-b) may waste huge bandwidth and may be suspected of being used for the purpose of attack.
