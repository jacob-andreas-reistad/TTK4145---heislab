# Elevator Instructions
1) Define system parameters in config.go:
    - number of elevators 
    - number of floors
    - etc. (see config.go)
2) Run the command from terminal: go run main.go -id=x -port=xxxxx
3) Shazam!


To get the hall_request_assigner executable, run these commands (make sure folders are correct so they end up in the assigner folder): 
wget https://github.com/TTK4145/Project-resources/releases/latest/download/hall_request_assigner
mkdir -p /home/student/Documents/gr77tir/assigner/executables
mv hall_request_assigner /home/student/Documents/gr77tir/assigner/executables/
chmod a+x /home/student/Documents/gr77tir/assigner/executables/hall_request_assigner
