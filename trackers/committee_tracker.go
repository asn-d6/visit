/// This module is responsible for tracking committees and correlating them
/// with attestations. It's also responsible for informing the activity tracker
/// about the status of validators.

package trackers

import (
	"errors"
	"fmt"

	"github.com/protolambda/eth2api"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/beacon/phase0"
)

// The core of this module: Tracks committees and provides useful functions for
// investigating them and correlating them with attestations
type CommitteeTracker struct {
	// Tracks committees per epoch
	// { Slot #123123 : { Committee #1,
	//                     Committee #2 },
	//   Slot #441234 : { Committee #1123, ... } }
	tracker map[common.Slot][]eth2api.Committee
}

func InitCommitteeTracker() *CommitteeTracker {
	var committeeTracker CommitteeTracker
	committeeTracker.tracker = make(map[common.Slot][]eth2api.Committee)
	return &committeeTracker
}

// Register new committee information to the tracker
func (ct *CommitteeTracker) RegisterCommittees(committees []eth2api.Committee) {
	// Just used to reduce the amount of debug logging going on
	var seenSlot = map[common.Slot]bool{}

	for _, c := range committees {
		if !seenSlot[c.Slot] { // some debug logging
			fmt.Printf("\tGot fresh committee info: registering committee %d for slot #%d\n", c.Index, c.Slot)
			seenSlot[c.Slot] = true
		}

		// Register the committee for that slot
		ct.tracker[c.Slot] = append(ct.tracker[c.Slot], c)
	}
}

// Check whether we are tracking the committes for `slot`
func (ct *CommitteeTracker) CommitteesAreKnownForSlot(slot common.Slot) bool {
	if ct.tracker[slot] == nil {
		return false
	}
	return true
}

// Return the committee for the given index/slot, or an error if it can't be found
func (ct *CommitteeTracker) getCommitteeFromIndex(index common.CommitteeIndex, slot common.Slot) (*eth2api.Committee, error) {
	for _, c := range ct.tracker[slot] {
		if c.Index == index {
			return &c, nil
		}
	}
	return nil, errors.New("No committee found")
}

// Given an attestation (found in `blockSlot`), handle it and register the
// validators with the activity tracker
func (ct *CommitteeTracker) handleAttestation(att phase0.Attestation, blockSlot common.Slot) {
	committee, err := ct.getCommitteeFromIndex(att.Data.Index, att.Data.Slot)
	if err != nil {
		fmt.Printf("[*] Debugging attestation for committee #%d and slot #%d...\n", att.Data.Index, att.Data.Slot)
		panic("didnt know the committee")
	}

	//	fmt.Printf("[*] Handling attestation (%d bits set) for committee #%d (%d validators) and slot #%d (bitfield: %s)\n",
	//		att.AggregationBits.OnesCount(), committee.Index, len(committee.Validators), att.Data.Slot, att.AggregationBits)

	// Sanity check: The attestation we are handling should be composed using
	// the committee composition we are tracking. These two must not get desynced.
	if !ct.CommitteesAreKnownForSlot(att.Data.Slot) {
		fmt.Printf("[*] Debugging attestation for committee #%d and slot #%d...\n", att.Data.Index, att.Data.Slot)
		panic("after all... we didn't really visit our committees...")
	}

	//	fmt.Printf("[*] Validators for committee #%d: %v\n", committee.Index, committee.Validators)

	// Process validators in the committee sequentially and cross-reference
	// them with the aggregated bitfield
	for i, valIndex := range committee.Validators {
		var is_present bool = att.AggregationBits.GetBit(uint64(i))
		registerValidatorPresense(valIndex, att.Data.Slot, blockSlot, is_present)
	}
}

// Handle all the `attestations` of `blockSlot`
func (ct *CommitteeTracker) HandleAttestations(attestations []phase0.Attestation, blockSlot common.Slot) {
	registerNewBlock(blockSlot)

	for _, att := range attestations {
		ct.handleAttestation(att, blockSlot)
	}
}
