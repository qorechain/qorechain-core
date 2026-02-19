package keeper

import (
	"context"
	"math"
	"sync"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/ai/types"
)

// FeeOptimizer implements predictive fee optimization per the whitepaper.
// It predicts network congestion and suggests optimal fees for desired confirmation times.
type FeeOptimizer struct {
	congestionPredictor *CongestionPredictor
	feeEstimator        *FeeEstimator
}

// NewFeeOptimizer creates a new fee optimizer.
func NewFeeOptimizer() *FeeOptimizer {
	return &FeeOptimizer{
		congestionPredictor: NewCongestionPredictor(),
		feeEstimator:        NewFeeEstimator(),
	}
}

// EstimateFee returns a fee estimate for the given urgency level.
func (fo *FeeOptimizer) EstimateFee(_ context.Context, urgency string) (*types.FeeEstimate, error) {
	// Get current congestion from predictor
	currentCongestion := fo.congestionPredictor.CurrentCongestion()
	predictedCongestion := fo.congestionPredictor.PredictCongestion(10) // Next 10 blocks

	// Estimate fee based on urgency and congestion
	estimate := fo.feeEstimator.Estimate(urgency, currentCongestion, predictedCongestion)

	return estimate, nil
}

// RecordBlockStats feeds new block data into the congestion predictor.
func (fo *FeeOptimizer) RecordBlockStats(height int64, txCount int, gasUsed uint64, maxGas uint64) {
	fo.congestionPredictor.RecordBlock(height, txCount, gasUsed, maxGas)
}

// ---- Congestion Predictor ----

// CongestionPredictor uses exponential moving average of block utilization
// to predict future congestion levels.
type CongestionPredictor struct {
	mu         sync.RWMutex
	history    []blockStats
	maxHistory int
	ema        float64 // Exponential moving average of congestion
	alpha      float64 // EMA smoothing factor
}

type blockStats struct {
	height    int64
	txCount   int
	gasUsed   uint64
	maxGas    uint64
	timestamp time.Time
}

func NewCongestionPredictor() *CongestionPredictor {
	return &CongestionPredictor{
		history:    make([]blockStats, 0, 100),
		maxHistory: 100,
		ema:        0.1, // Start with low congestion assumption
		alpha:      0.2, // EMA smoothing factor (higher = more responsive)
	}
}

// RecordBlock records new block statistics for congestion tracking.
func (cp *CongestionPredictor) RecordBlock(height int64, txCount int, gasUsed uint64, maxGas uint64) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	bs := blockStats{
		height:    height,
		txCount:   txCount,
		gasUsed:   gasUsed,
		maxGas:    maxGas,
		timestamp: time.Now(),
	}

	cp.history = append(cp.history, bs)
	if len(cp.history) > cp.maxHistory {
		cp.history = cp.history[1:]
	}

	// Update EMA with new block's congestion
	var blockCongestion float64
	if maxGas > 0 {
		blockCongestion = float64(gasUsed) / float64(maxGas)
	} else {
		// Fallback: use TX count heuristic
		blockCongestion = math.Min(float64(txCount)/100.0, 1.0)
	}

	cp.ema = cp.alpha*blockCongestion + (1-cp.alpha)*cp.ema
}

// CurrentCongestion returns the current smoothed congestion level (0.0 to 1.0).
func (cp *CongestionPredictor) CurrentCongestion() float64 {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return cp.ema
}

// PredictCongestion predicts congestion for the next N blocks using trend analysis.
func (cp *CongestionPredictor) PredictCongestion(blocksAhead int) float64 {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	if len(cp.history) < 5 {
		return cp.ema // Not enough data for trend analysis
	}

	// Compute trend: compare recent EMA to older EMA
	recent := cp.history[len(cp.history)-5:]
	var recentCongestion, olderCongestion float64
	for _, bs := range recent {
		if bs.maxGas > 0 {
			recentCongestion += float64(bs.gasUsed) / float64(bs.maxGas)
		}
	}
	recentCongestion /= float64(len(recent))

	olderStart := 0
	if len(cp.history) > 10 {
		olderStart = len(cp.history) - 10
	}
	olderSlice := cp.history[olderStart : len(cp.history)-5]
	if len(olderSlice) > 0 {
		for _, bs := range olderSlice {
			if bs.maxGas > 0 {
				olderCongestion += float64(bs.gasUsed) / float64(bs.maxGas)
			}
		}
		olderCongestion /= float64(len(olderSlice))
	}

	// Trend per block
	trend := (recentCongestion - olderCongestion) / 5.0

	// Project forward (with dampening)
	predicted := cp.ema + trend*float64(blocksAhead)*0.5
	return math.Max(0.0, math.Min(predicted, 1.0))
}

// ---- Fee Estimator ----

// FeeEstimator computes fee suggestions based on congestion and urgency.
type FeeEstimator struct {
	baseFee uint64 // Base fee in uqor (500 uqor minimum)
}

func NewFeeEstimator() *FeeEstimator {
	return &FeeEstimator{
		baseFee: 500, // 0.0005 QOR
	}
}

// Estimate computes a fee estimate for the given urgency and congestion levels.
func (fe *FeeEstimator) Estimate(urgency string, currentCongestion, predictedCongestion float64) *types.FeeEstimate {
	// Urgency multipliers
	var urgencyMultiplier float64
	var estimatedBlocks int

	switch urgency {
	case "fast":
		urgencyMultiplier = 2.0
		estimatedBlocks = 1
	case "normal":
		urgencyMultiplier = 1.0
		estimatedBlocks = 3
	case "slow":
		urgencyMultiplier = 0.5
		estimatedBlocks = 10
	default:
		urgencyMultiplier = 1.0
		estimatedBlocks = 3
	}

	// Congestion multiplier: 1.0 at 0% congestion, up to 5.0 at 100% congestion
	congestionMultiplier := 1.0 + currentCongestion*4.0

	// If predicted congestion is increasing, add premium
	if predictedCongestion > currentCongestion {
		congestionMultiplier *= 1.0 + (predictedCongestion-currentCongestion)*2.0
	}

	// Calculate suggested fee
	suggestedFee := uint64(float64(fe.baseFee) * urgencyMultiplier * congestionMultiplier)

	// Minimum fee floor
	if suggestedFee < fe.baseFee {
		suggestedFee = fe.baseFee
	}

	// Adjust estimated blocks based on congestion
	if currentCongestion > 0.8 && urgency != "fast" {
		estimatedBlocks = int(float64(estimatedBlocks) * (1.0 + currentCongestion))
	}

	// Confidence based on data quality
	confidence := 0.9
	if currentCongestion > 0.8 {
		confidence = 0.7 // Less confident in high congestion
	}

	return &types.FeeEstimate{
		SuggestedFee:        sdk.NewCoin("uqor", sdkmath.NewIntFromUint64(suggestedFee)),
		EstimatedBlocks:     estimatedBlocks,
		CurrentCongestion:   currentCongestion,
		PredictedCongestion: predictedCongestion,
		Confidence:          confidence,
	}
}
