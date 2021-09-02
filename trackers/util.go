package trackers

import "github.com/protolambda/zrnt/eth2/beacon/common"

// Helper function from the friendly spec
func ComputeEpochAtSlot(slot common.Slot) common.Epoch {
	return common.Epoch(slot / 32)
}

// Helper function from the friendly spec
func SlotBelongsToEpoch(slot common.Slot, epoch common.Epoch) bool {
	if (ComputeEpochAtSlot(slot)) != epoch {
		return false
	}
	return true
}

func ComputeStartSlotAtEpoch(epoch common.Epoch) common.Slot {
	return common.Slot(epoch * 32)
}

func ComputeSlotIndexWithinEpoch(slot common.Slot) int {
	epoch := ComputeEpochAtSlot(slot)
	return int(slot - ComputeStartSlotAtEpoch(epoch))
}

// Return the first epoch after `slot`. So if we give it the 17th slot of epoch
// #7, this should return #8. OTOH if we give it the 0th slot of epoch #9 it
// should return #9 since it's just starting.
func firstEpochAfterSlot(slot common.Slot) common.Epoch {
	epoch := ComputeEpochAtSlot(slot)

	slot_idx := ComputeSlotIndexWithinEpoch(slot)
	if (slot_idx == 0) {
		return epoch
	} else {
		return epoch+1
	}
}

// Return the last epoch before `slot`. We always return the previous epoch.
func lastEpochBeforeSlot(slot common.Slot) common.Epoch {
	epoch := ComputeEpochAtSlot(slot)
	return epoch-1
}

