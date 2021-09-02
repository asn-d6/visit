/// This module is responsible for tracking the activity and presense of
/// validators. It's also responsible for writing the validator status to the
/// database. Some people consider it the bread and butter of this novel
/// software stack.

package trackers

import (
	"fmt"

	"github.com/asn-d6/visit/db"
	"github.com/protolambda/zrnt/eth2/beacon/common"
)

const (
	// Magic number that signals a missing validator
	VALIDATOR_MISSING_MAGIC = 65535
)

// Tracks the activity of validators per epoch. Maps epochs to validators, and
// validators to inclusion distance.
//
// { Epoch #123123 : { Validator #1 : 12
//                     Validator #2 : 14
//                     Validator #132 : 0 }
//   Epoch #123511 : { Validator #6 :48
//                     Validator #8 : 23 ... } }
//
// TODO: Another interesting data point to keep would be the *slot* that the
// validator was supposed to attest to in each epoch; to see if the slot was
// missing
var validatorActivityTracker = map[common.Epoch]map[common.ValidatorIndex]int{}

// Tracks which validators are interesting for our analysis (only validators
// that have been slow or missing are interesting to us... we are weird)
//
// XXX False positive: Inclusion distance can be high even in "normal"
// circumstances if some blocks fail to get published, since the next block is
// gonna include attestations about old blocks.
var interestingValidators = map[common.ValidatorIndex]bool{}

var numInterestingValidators int

////////////////////////////////////////////////////////////////////////////

// We just learned about the presense of validator `index` from an attestation
// to slot `attestationSlot` that was found in block `blockSlot`.
// The validator was either present or not, depending on the value of `is_present`
//
// XXX eek this code smells horrible
func registerValidatorPresense(valIndex common.ValidatorIndex, attestationSlot common.Slot, blockSlot common.Slot, is_present bool) {
	epoch := ComputeEpochAtSlot(attestationSlot)
	if validatorActivityTracker[epoch] == nil { // initialize map if needed
		validatorActivityTracker[epoch] = make(map[common.ValidatorIndex]int)
	}

	// Inclusion distance is how far back in time is the slot that this
	// validator is attesting for compared to the current block (max distance
	// is 64 slots)
	inclusion_distance := int(blockSlot - attestationSlot)

	if is_present {
		// The inclusion distance shouldn't increase
		if validatorActivityTracker[epoch][valIndex] != 0 && inclusion_distance >= validatorActivityTracker[epoch][valIndex] {
			return
		}

		validatorActivityTracker[epoch][valIndex] = inclusion_distance

		if inclusion_distance > 1 { // flag slow validators as interesting
			interestingValidators[valIndex] = true
			numInterestingValidators++
		}

		// Validators with optimal inclusion distance could also be flagged as
		// "interesting" if there are two attestations for the same slot in the
		// block. The first one does not include them but the second one
		// includes them. So deflag them here.
		if inclusion_distance == 1 && interestingValidators[valIndex] {
			interestingValidators[valIndex] = false
			numInterestingValidators--
		}
//		fmt.Printf("\tValidator #%d distance: %d (cur block %d / att %d) (%d interesting)\n", valIndex, validatorActivityTracker[epoch][valIndex], blockSlot, attestationSlot, numInterestingValidators)
	} else {
		// If the validator has already been flagged as missing, or we have
		// seen her before in a previous attestation, don't flag her as missing.
		if validatorActivityTracker[epoch][valIndex] != 0 {
			return
		}

//		fmt.Printf("\tValidator #%d marked as missing (%d interesting)\n", valIndex, numInterestingValidators)

		validatorActivityTracker[epoch][valIndex] = VALIDATOR_MISSING_MAGIC
		interestingValidators[valIndex] = true
		numInterestingValidators++
	}
}

////////////////////////////////////////////////////////////////////////////

// TODO Track first fully seen epoch and last epoch. All between is good.

// Track slots seen in this epoch. Used to make sure we only dump metrics about
// epochs we have completely seen
var firstSlotSeen common.Slot
var lastSlotSeen common.Slot


// A new block was processed. Register it for the purposes of figuring out how
// many epochs we've seen
func registerNewBlock(slot common.Slot) {
	if firstSlotSeen == 0 {
		firstSlotSeen = slot
	}

	lastSlotSeen = slot

	// TODO this function should be called after the block is processed, and
	// when we move past an epoch, it should spawn a goroutine that dumps the
	// activity tracker of the epoch that just passed to the database
}

func DumpActivityTracker() {
	fmt.Printf("Dumping the data! Brace for impact.\n")

	// Track which epochs have been fully seen (we were here in their beginning and end)
	var fullySeenEpochs []common.Epoch
	first_epoch := firstEpochAfterSlot(firstSlotSeen)
	last_epoch := lastEpochBeforeSlot(lastSlotSeen)
	for epoch := first_epoch ; epoch <= last_epoch ; epoch++ {
		fullySeenEpochs = append(fullySeenEpochs, epoch)
	}

	fmt.Printf("We have %d interesting validators over %d fully seen epochs: %v\n", numInterestingValidators, len(fullySeenEpochs),fullySeenEpochs)

	db := db.InitDatabase()
	defer db.Close()

	var i int
	for _, epoch := range fullySeenEpochs {
		validatorMap := validatorActivityTracker[epoch]
		for validator, state := range validatorMap {
			if interestingValidators[validator] == false {
				continue
			}
			if i%1000 == 0 {
				fmt.Printf("Dumping %dth validator...\n", i)
			}
			db.RegisterAttestation(int(validator), int(epoch), state)
			i++
		}
	}
}
