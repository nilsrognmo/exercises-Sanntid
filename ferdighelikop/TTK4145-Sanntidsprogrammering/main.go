package main

import (
	"TTK4145-Heislab/Network-go/network/bcast"
	"TTK4145-Heislab/Network-go/network/peers"
	"TTK4145-Heislab/communication"
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/driver-go/elevio"
	"TTK4145-Heislab/single_elevator"
	"TTK4145-Heislab/worldview"
	"flag"
	"fmt"
	"time"
)

func main() {
	fmt.Println("Elevator System Starting...")

	// map1 := make(map[string]bool)
	// map1["A"] = true
	// fmt.Println("Before copy: ", map1)

	// map2 := map1
	// map2["A"] = false

	// fmt.Println("after: map1: ", map1, " map2: ", map2)

	// i := 1

	// j := &i
	// *j = 2

	// fmt.Println("i: ", i, " j: ", *j)

	// return

	idflag := flag.String("id", "default_id", "Id of this peer")
	elevioPortFlag := flag.String("ePort", "15657", "Port for elevio")
	flag.Parse()
	elevatorID := *idflag
	fmt.Println("My id: ", elevatorID)
	fmt.Println("Elevio port: ", *elevioPortFlag)

	// Initialize elevator hardware
	elevio.Init("localhost:"+*elevioPortFlag, configuration.NumFloors) //elevator - har laget ports i configuration

	// Communication channels
	newOrderChannel := make(chan single_elevator.Orders, configuration.Buffer)
	completedOrderChannel := make(chan elevio.ButtonEvent, configuration.Buffer)
	buttonPressedChannel := make(chan elevio.ButtonEvent, configuration.Buffer)
	WorldViewTXChannel := make(chan worldview.WorldView, configuration.Buffer)
	WorldViewRXChannel := make(chan worldview.WorldView, configuration.Buffer)
	IDPeersChannel := make(chan []string)
	peerUpdateChannel := make(chan peers.PeerUpdate)
	elevatorStateChannel := make(chan single_elevator.Elevator, configuration.Buffer)
	elevatorTimeoutTimer := time.NewTimer(5 * time.Second)
	//initDirection := worldview.DetermineInitialDirection(WorldViewRXChannel, elevatorID)
	//initDirection := worldview.FetchInitialDirection(WorldViewRXChannel, elevatorID)
	//var initDirection elevio.MotorDirection

	go bcast.Transmitter(configuration.BroadcastPort, WorldViewTXChannel)
	go bcast.Receiver(configuration.BroadcastPort, WorldViewRXChannel)

	enableTransmit := make(chan bool)
	go peers.Transmitter(configuration.PeersPort, elevatorID, enableTransmit) //vi sender aldri noe inn i peers: transmitEnable <-chan bool
	go peers.Receiver(configuration.PeersPort, peerUpdateChannel)

	// Start FSM
	go elevio.PollButtons(buttonPressedChannel)
	//har started polling på obstruction, floorsensor, stopbutton i FSM

	var initDirection elevio.MotorDirection = elevio.MD_Down

	select {
	case worldView := <-WorldViewRXChannel:
		if worldView.ElevatorStatusList != nil {
			if _, ok := worldView.ElevatorStatusList[elevatorID]; ok {
				//initDirection = elevio.MotorDirection(worldView.ElevatorStatusList[elevatorID].Elev.Direction)
				if worldView.ElevatorStatusList[elevatorID].Elev.Direction == single_elevator.Down {
					initDirection = elevio.MD_Down
				} else {
					initDirection = elevio.MD_Up
				}
			}
		}
	case <-time.After(100 * time.Millisecond):
		initDirection = elevio.MD_Down
	}

	go single_elevator.SingleElevator(newOrderChannel, completedOrderChannel, elevatorStateChannel, initDirection)
	go communication.CommunicationHandler(elevatorID, peerUpdateChannel, IDPeersChannel)
	go worldview.WorldViewManager(elevatorID, WorldViewTXChannel, WorldViewRXChannel, buttonPressedChannel, newOrderChannel, completedOrderChannel, IDPeersChannel, elevatorStateChannel, elevatorTimeoutTimer)

	select {}
}

/*
Network module
- UDP connection (packet loss) - packet sending and receiving (message format - JSON?) **concurrency
- Broadcasting (peer addresses, goroutine to periodically broadcast the elevator's state to all other peers)
- Message handling (message serialization/deserialization)

Peer to Peer module
- peer discovery
- message exchange
- peer failures
- synchronize the states

Assigner/Decision making module (cost function)
Fault Tolerance
*/
