package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const (
	numOfBarbers    = 2                // Number of barbers
	numOfChairs     = 5                // Number of chairs in the waiting room
	openingTime     = 8 * time.Hour    // Opening time of the shop
	closingTime     = 20 * time.Hour   // Closing time of the shop
	clientInterval  = 2 * time.Hour    // Interval at which clients arrive
	haircutDuration = 30 * time.Minute // Duration of a haircut
)

var (
	wg          sync.WaitGroup
	clients     = make(chan struct{}, numOfChairs) // Signaling channel for client arrival
	barberReady = make(chan bool, numOfBarbers)    // Signaling channel for barber readiness
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// Start the barber goroutines
	for i := 0; i < numOfBarbers; i++ {
		wg.Add(1)
		barberReady <- true // Initialize all barbers as ready to work
		go barber(i)
	}

	// Open the shop
	fmt.Println("Barbershop is open!")

	// Start the clock
	go clock()

	// Wait for all barbers to finish their shift
	wg.Wait()

	fmt.Println("Barbershop is closed.")
}

func clock() {
	defer close(clients)

	// Run the shop from opening time to closing time
	for currentTime := openingTime; currentTime < closingTime; currentTime += clientInterval {
		// Simulate arrival of a client
		time.Sleep(clientInterval)
		select {
		case clients <- struct{}{}:
		default:
			// No available chairs, client leaves
			fmt.Println("Client left due to no available chairs.")
		}
	}
}

func barber(id int) {
	defer wg.Done()

	fmt.Printf("Barber %d started working\n", id+1)

	for {
		select {
		case <-clients:
			// A client has arrived
			fmt.Printf("Barber %d is cutting hair\n", id+1)
			time.Sleep(haircutDuration)
			fmt.Printf("Barber %d finished cutting hair\n", id+1)

		default:
			// No clients, barber goes to sleep
			fmt.Printf("Barber %d is sleeping\n", id+1)
			<-barberReady // Wait until ready to work again
		}
	}
}
