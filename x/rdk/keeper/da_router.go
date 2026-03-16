//go:build proprietary

package keeper

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strconv"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/rdk/types"
)

// --- DA Blob helpers ---

func daBlobKey(rollupID string, blobIndex uint64) []byte {
	return append(types.DABlobPrefix, []byte(rollupID+"/"+strconv.FormatUint(blobIndex, 10))...)
}

func latestDAKey(rollupID string) []byte {
	return append(types.LatestDAPrefix, []byte(rollupID)...)
}

// SubmitDABlob routes a DA blob to the appropriate backend.
func (k Keeper) SubmitDABlob(ctx sdk.Context, blob types.DABlob) (*types.DACommitment, error) {
	// Validate rollup is active
	rollup, err := k.GetRollup(ctx, blob.RollupID)
	if err != nil {
		return nil, err
	}
	if rollup.Status != types.RollupStatusActive {
		return nil, types.ErrRollupNotActive
	}

	// Check blob size
	params := k.GetParams(ctx)
	if uint64(len(blob.Data)) > params.MaxDABlobSize {
		return nil, types.ErrDABlobTooLarge
	}

	backend := rollup.DABackend

	switch backend {
	case types.DANative, types.DABoth:
		// Store blob in KVStore
		commitment := k.storeNativeBlob(ctx, blob)

		if backend == types.DABoth {
			// Log Celestia stub warning for the "both" backend
			k.logger.Warn("Celestia DA backend is stubbed in v1.3.0, only native blob stored", "rollup_id", blob.RollupID)
		}

		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventDABlobStored,
			sdk.NewAttribute("rollup_id", blob.RollupID),
			sdk.NewAttribute("blob_index", strconv.FormatUint(blob.BlobIndex, 10)),
			sdk.NewAttribute("backend", string(types.DANative)),
			sdk.NewAttribute("size", strconv.FormatUint(uint64(len(blob.Data)), 10)),
		))

		return commitment, nil

	case types.DACelestia:
		return nil, types.ErrCelestiaDAStubed

	default:
		return nil, fmt.Errorf("unknown DA backend: %s", backend)
	}
}

// storeNativeBlob stores a blob in KVStore and computes a SHA-256 commitment.
func (k Keeper) storeNativeBlob(ctx sdk.Context, blob types.DABlob) *types.DACommitment {
	// Compute commitment (SHA-256 hash of data)
	hash := sha256.Sum256(blob.Data)
	blob.Commitment = hash[:]
	blob.Height = ctx.BlockHeight()
	blob.StoredAt = ctx.BlockTime().UTC()
	blob.Pruned = false

	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(blob)
	if err != nil {
		k.logger.Error("failed to marshal DA blob", "error", err)
		return nil
	}
	store.Set(daBlobKey(blob.RollupID, blob.BlobIndex), bz)

	// Update latest DA pointer (store only the blob index)
	indexBz := make([]byte, 8)
	binary.LittleEndian.PutUint64(indexBz, blob.BlobIndex)
	store.Set(latestDAKey(blob.RollupID), indexBz)

	return &types.DACommitment{
		RollupID:  blob.RollupID,
		BlobIndex: blob.BlobIndex,
		Backend:   types.DANative,
		Hash:      hash[:],
		Size:      uint64(len(blob.Data)),
		Confirmed: true,
	}
}

// GetDABlob retrieves a DA blob from KVStore.
func (k Keeper) GetDABlob(ctx sdk.Context, rollupID string, blobIndex uint64) (*types.DABlob, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(daBlobKey(rollupID, blobIndex))
	if bz == nil {
		return nil, types.ErrDABlobNotFound
	}
	var blob types.DABlob
	if err := json.Unmarshal(bz, &blob); err != nil {
		return nil, fmt.Errorf("failed to unmarshal DA blob: %w", err)
	}
	return &blob, nil
}

// GetLatestDABlob retrieves the latest DA blob for a rollup by reading
// the stored blob index from the latest-DA pointer and fetching the full blob.
func (k Keeper) GetLatestDABlob(ctx sdk.Context, rollupID string) (*types.DABlob, error) {
	store := ctx.KVStore(k.storeKey)
	indexBz := store.Get(latestDAKey(rollupID))
	if indexBz == nil || len(indexBz) < 8 {
		return nil, types.ErrDABlobNotFound
	}
	blobIndex := binary.LittleEndian.Uint64(indexBz)
	return k.GetDABlob(ctx, rollupID, blobIndex)
}

// maxPrunePerBlock caps the number of blobs pruned per EndBlocker call
// to bound the work done per block and avoid gas spikes.
const maxPrunePerBlock = 100

// PruneExpiredBlobs iterates DA blobs and marks as pruned if past retention period.
// Caps at maxPrunePerBlock per call to avoid excessive per-block work.
func (k Keeper) PruneExpiredBlobs(ctx sdk.Context) (uint64, error) {
	params := k.GetParams(ctx)
	currentHeight := ctx.BlockHeight()
	var pruned uint64

	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.DABlobPrefix)
	defer iter.Close()

	for ; iter.Valid() && pruned < maxPrunePerBlock; iter.Next() {
		var blob types.DABlob
		if err := json.Unmarshal(iter.Value(), &blob); err != nil {
			continue
		}
		if blob.Height+int64(params.BlobRetentionBlocks) < currentHeight {
			store.Delete(iter.Key())
			pruned++

			ctx.EventManager().EmitEvent(sdk.NewEvent(
				types.EventDABlobPruned,
				sdk.NewAttribute("rollup_id", blob.RollupID),
				sdk.NewAttribute("blob_index", strconv.FormatUint(blob.BlobIndex, 10)),
			))
		}
	}

	if pruned > 0 {
		k.logger.Info("pruned expired DA blobs", "count", pruned)
	}
	return pruned, nil
}
