Network module for Go (UDP broadcast only)
==========================================

See [`main.go`](main.go) for usage example. The code is runnable with just `go run main.go`

Add these lines to your `go.mod` file:
```
require network v0.0.0
replace network => ./network
```
Where `./network` is the relative path to this folder, after you have downloaded it.


Features
--------

Channel-in/channel-out pairs of (almost) any custom or built-in data type can be supplied to a pair of transmitter/receiver functions. Data sent to the transmitter function is automatically serialized and broadcast on the specified port. Any messages received on the receiver's port are de-serialized (as long as they match any of the receiver's supplied channel data types) and sent on the corresponding channel. See [bcast.Transmitter and bcast.Receiver](bcast/bcast.go).

Peers on the local network can be detected by supplying your own ID to a transmitter and receiving peer updates (new, current, and lost peers) from the receiver. See [peers.Transmitter and peers.Receiver](peers/peers.go).

Finding your own local IP address can be done with the [LocalIP](localip/localip.go) convenience function, but only when you are connected to the internet.








