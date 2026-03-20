# Elevator Instructions
1) Define system parameters in config.go:
    - number of elevators 
    - number of floors
    - etc. (see config.go)
2) Run the command from terminal: go run main.go -id=x -port=xxxxx
3) Shazam!

## Project Structure
The system runs one instance per elevator. Each instance communicates with the others over UDP,
maintaining a shared common state. The main modules are:

- `elevator/` — finite state machine controlling motor, doors, and orders
- `distributor/` — network synchronization and consensus between elevators
- `assigner/` — delegates order assignment to an external cost function executable
- `network/bcast/` — UDP broadcast transmitter and receiver
- `network/peers/` — detects elevators joining and leaving the network
- `network/localip/` — resolves the local IP address for networking
- `network/conn/` — platform-specific UDP socket configuration
- `elevio/` — hardware I/O driver; polls buttons, floor sensor, obstruction and stop switches
- `lights/` — updates hall and cab button lamps on the panel
- `config/` — system-wide constants (number of floors, elevators, timeouts)
- `packetloss/` — utility for simulating network packet loss; included to simplify setup during FAT-testing
- 


### Module Communication
All modules communicate via Go channels, coordinated in `main.go`:

- `elevio` → `elevator`: floor sensor and obstruction events (internal polling goroutines)
- `elevio` → `distributor`: button press events (`buttonEventCh`, internal to synchronizer)
- `elevator` → `distributor`: local state updates (`stateUpdateCh`) and completed orders (`orderDoneCh`)
- `distributor` ↔ `network/bcast`: common state broadcast over UDP (`networkTx`, `networkRx`)
- `network/peers` → `distributor`: peer join/leave notifications (`peersRx`)
- `distributor` → `main`: confirmed common state (`csConfirmedCh`)
- `main` → `assigner`: confirmed common state for cost calculation
- `assigner` → `main`: assigned orders for this elevator (`newOrderCh`)
- `main` → `elevator`: new orders to execute (`newOrderCh`)
- `main` → `lights`: confirmed common state for panel light updates