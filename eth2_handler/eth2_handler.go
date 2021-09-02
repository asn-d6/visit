/// This module takes care of the communication between this script and
/// Ethereum using eth2api. It passes the network data to the committee
/// tracker for further investigation.

package eth2_handler

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/asn-d6/visit/trackers"

	"github.com/protolambda/eth2api"
	"github.com/protolambda/eth2api/client/beaconapi"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/configs"

	"github.com/protolambda/zrnt/eth2/beacon/phase0"
)

type Eth2Handler struct {
	// Various information for eth2api to function
	client     *eth2api.Eth2HttpClient
	ctx        context.Context
	genesis    eth2api.GenesisResponse
	forkDigest common.ForkDigest
	spec       *common.Spec

	// Our trusted committee tracker. Keeps track of committees so that we can
	// correlate them with attestations when needed
	committeeTracker *trackers.CommitteeTracker
}

const (
	blockHead = eth2api.BlockHead
	stateHead = eth2api.StateHead
)

func InitEth2Handler() *Eth2Handler {
	// Make an HTTP client (reuse connections!)
	client := &eth2api.Eth2HttpClient{
		Addr: "http://" + os.Args[1],
		Cli: &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 123,
			},
			Timeout: 40 * time.Second,
		},
		Codec: eth2api.JSONCodec{},
	}

	//// e.g. cancel requests with a context.WithTimeout/WithCancel/WithDeadline
	ctx := context.Background()

	var genesis eth2api.GenesisResponse
	if exists, err := beaconapi.Genesis(ctx, client, &genesis); !exists {
		fmt.Println("chain did not start yet")
		os.Exit(1)
	} else if err != nil {
		fmt.Println("failed to get genesis", err)
		os.Exit(1)
	}

	spec := configs.Mainnet
	// or load testnet config info from a YAML file
	// yaml.Unmarshal(data, &spec.Config)

	// every fork has a digest. Blocks are versioned by name in the API,
	// but wrapped with digest info in ZRNT to do enable different kinds of processing
	forkDigest := common.ComputeForkDigest(spec.ALTAIR_FORK_VERSION, genesis.GenesisValidatorsRoot)

	return &Eth2Handler{
		client:           client,
		ctx:              ctx,
		genesis:          genesis,
		forkDigest:       forkDigest,
		spec:             spec,
		committeeTracker: trackers.InitCommitteeTracker(),
	}
}

func getAttestationsFromBlock(signedBlock phase0.SignedBeaconBlock) []phase0.Attestation {
	return signedBlock.Message.Body.Attestations
}

// Make sure we know all committees referenced by these attestations
func (h *Eth2Handler) FetchCommitteeInfoIfNeeded(attestations []phase0.Attestation) {
	for _, att := range attestations {
		if !h.committeeTracker.CommitteesAreKnownForSlot(att.Data.Slot) {
			// Fetch committees for the entire current epoch
			fmt.Printf("[!] Fetched block with attestations for slot #%d but we don't have"+
				" committee info for it. Fetching...\n", att.Data.Slot)
			h.getCommittees(nil, nil)

			// If the above committee fetch was not sufficient (e.g. because
			// the attestation was refering to a previous epoch), fetch the
			// specific slot committee
			if !h.committeeTracker.CommitteesAreKnownForSlot(att.Data.Slot) {
				fmt.Printf("[!] Fetching committees specifically for that slot...\n")
				epoch := trackers.ComputeEpochAtSlot(att.Data.Slot)
				h.getCommittees(&att.Data.Slot, &epoch)
			}
		}
	}
}

// Attempt to fetch and process attestations of the block at `blockNumber` (get 'head' if it's zero)
//
// If the block was fetched and handled, return its block number, else return 0.
func (h *Eth2Handler) FetchAndProcessBlock(blockNumber int) int {
	var signedBlock phase0.SignedBeaconBlock
	var exists bool
	var err error

	if blockNumber != 0 {
		fmt.Printf("[*] Attempting to fetch block for slot #%d\n", blockNumber)
		exists, err = beaconapi.Block(h.ctx, h.client, eth2api.BlockIdSlot(common.Slot(blockNumber)), &signedBlock)
	} else {
		fmt.Printf("[*] Attempting to fetch the latest block\n")
		exists, err = beaconapi.Block(h.ctx, h.client, blockHead, &signedBlock)
	}

	if !exists { // block not here yet. it's ok we will retry
		return 0
	}

	if err != nil { // unrecoverable error. time to panic hard.
		panic(err)
	}

	// Add metadata to signed block so that we can access its fields
	attestations := getAttestationsFromBlock(signedBlock)

	h.FetchCommitteeInfoIfNeeded(attestations)

	epoch := trackers.ComputeEpochAtSlot(signedBlock.Message.Slot)
	fmt.Printf("[*] Fetched block for slot #%d (slot %d of epoch #%d) (#%d attestations)\n", signedBlock.Message.Slot,
		trackers.ComputeSlotIndexWithinEpoch(signedBlock.Message.Slot), epoch, len(attestations))

	h.committeeTracker.HandleAttestations(attestations, signedBlock.Message.Slot)

	return int(signedBlock.Message.Slot)
}

// Get the latest committee information and register them on the commitee tracker
//
// If `slot` and `epoch` are set, fetch that specific committee
func (h *Eth2Handler) getCommittees(slot *common.Slot, epoch *common.Epoch) {
	var committees []eth2api.Committee
	exists, err := beaconapi.EpochCommittees(h.ctx, h.client,
		stateHead,
		epoch, // epoch
		nil,   // committee index
		slot,  // slot
		&committees)

	// Handle errors
	if !exists {
		panic("commitee not found")
	} else if err != nil {
		panic(err)
	}

	h.committeeTracker.RegisterCommittees(committees)
}
