package types

import "fmt"

const (
	// DefaultMaxQueueTimeoutsPerBlock is how many queued calls one block
	// times out when params.MaxQueueTimeoutsPerBlock is zero, from the
	// crossvm-queue-height-index switch. MaxMaxQueueTimeoutsPerBlock is the
	// most governance may set. Calls beyond the bound time out in the
	// following blocks, oldest first; they never run once their timeout has
	// passed.
	DefaultMaxQueueTimeoutsPerBlock uint32 = 100
	MaxMaxQueueTimeoutsPerBlock     uint32 = 1_000

	// MaxQueueVisitsPerBlock bounds how many queued calls that have not
	// timed out one block looks at to find the ones it executes (at most
	// MaxAsyncCallsPerBlock), from the crossvm-queue-height-index switch. A
	// call the block cannot admit (its lane's budget is spent) is looked at
	// and left pending, so the bound is a few times the execution bound.
	MaxQueueVisitsPerBlock = 4 * MaxAsyncCallsPerBlock

	// MaxQueueScanPerBlock bounds how many index keys of calls that have not
	// timed out one block reads to take senders in turn, from the
	// crossvm-queue-sender-round-robin switch (two default queues; a key is
	// a few dozen bytes and nothing is decoded).
	MaxQueueScanPerBlock = 2 * int(DefaultMaxQueueSize)
)

// QueueTimeoutsPerBlock is the per-block timeout bound in force.
func (p Params) QueueTimeoutsPerBlock() uint32 {
	if p.MaxQueueTimeoutsPerBlock == 0 {
		return DefaultMaxQueueTimeoutsPerBlock
	}
	return p.MaxQueueTimeoutsPerBlock
}

func (p Params) validateQueueTimeouts() error {
	if p.MaxQueueTimeoutsPerBlock > MaxMaxQueueTimeoutsPerBlock {
		return fmt.Errorf("max queue timeouts per block %d is above %d", p.MaxQueueTimeoutsPerBlock, MaxMaxQueueTimeoutsPerBlock)
	}
	return nil
}
