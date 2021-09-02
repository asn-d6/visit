/// Entry point of visit.

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asn-d6/visit/eth2_handler"
	"github.com/asn-d6/visit/trackers"

	_ "github.com/mattn/go-sqlite3"
)

const (
	// How many slots should we monitor before aborting experiment?
	EXPERIMENT_DURATION_BLOCKS = 330

	// How many seconds we should wait before fetching a new block
	FETCH_BLOCK_SECONDS = 12

	// How many times we should try fetching a specific block before we give up
	SAME_BLOCK_RETRIES = 3
)

// Singleton that holds a bunch of state for our program
// XXX pretty useless atm
type Visit struct {
	// State required by eth2api to work
	eth2Handler *eth2_handler.Eth2Handler

	// Next slot to fech (we explicitly request specific block numbers
	// incrementally so that we don't miss any (if we are fetching too slow),
	// or continuously fetch the same one (if we are fetching too fast).
	nextSlotToFetch int
}

var fetchBlockTimer *time.Ticker

func lets_wrap_up() {
	fmt.Printf("***************** WRAPPING UP **********************\n")
	if fetchBlockTimer != nil {
		fetchBlockTimer.Stop()
	}
	trackers.DumpActivityTracker()
	os.Exit(0)
}

func (m *Visit) do_the_monitoring() {
	retry_counter := 0
	fetchBlockTimer = time.NewTicker(FETCH_BLOCK_SECONDS * time.Second)

	// Fetch the first block and mark its slot number
	handledSlot := m.eth2Handler.FetchAndProcessBlock(0)
	if handledSlot != 0 {
		m.nextSlotToFetch = handledSlot + 1
	}

	// Now incrementally fetch and process the next blocks
	var i uint = 1
	for {
		select {
		case <-fetchBlockTimer.C:
			// If experiment is done, dump the data, and let's go home
			if i >= EXPERIMENT_DURATION_BLOCKS {
				lets_wrap_up()
			}

			// Fetch the block and handle it (with retries if needed)
			handledSlot = m.eth2Handler.FetchAndProcessBlock(m.nextSlotToFetch)
			if handledSlot == 0 || handledSlot != m.nextSlotToFetch {
				// fetch failed (either 404 or wrong block returned): check if we should retry
				retry_counter++
				if retry_counter >= SAME_BLOCK_RETRIES {
					// We've tried too many times for the same block: give up
					fmt.Printf("[!] Giving up on slot #%d\n", m.nextSlotToFetch)
					m.nextSlotToFetch++
					retry_counter = 0
				}
			} else { // fetch success! let's move on
				m.nextSlotToFetch = handledSlot + 1
				retry_counter = 0
				i++
			}
		}
	}
}

func initialize_visit() *Visit {
	fmt.Println("[!] Initializing visit")

	eth2Handler := eth2_handler.InitEth2Handler()

	// Setup a sighandler
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		lets_wrap_up()
		os.Exit(1)
	}()

	visit := Visit{
		eth2Handler: eth2Handler,
	}
	return &visit
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Wrong usage! Try:\n\t./visit <ip:port>")
		os.Exit(1)
	}

	// Initialize the singleton thing that does everything
	visit := initialize_visit()

	visit.do_the_monitoring()
}
