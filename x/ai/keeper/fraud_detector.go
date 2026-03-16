//go:build proprietary

package keeper

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/qorechain/qorechain-core/x/ai/types"
)

// FraudDetector implements multi-layered fraud detection per the whitepaper:
// - Statistical Isolation Forest for anomaly scoring
// - Sequence analysis for pattern detection
// - Specific detectors: Sybil, DDoS, Flash Loan, Exploit
//
// Detection latency target: <500ms
// False positive rate target: <0.1%
type FraudDetector struct {
	isolationForest  *StatisticalIsolationForest
	sequenceAnalyzer *TxSequenceAnalyzer
	sybilDetector    *SybilDetector
	ddosDetector     *DDoSDetector
	flashLoanDetector *FlashLoanDetector
	exploitDetector  *ExploitDetector
}

// NewFraudDetector creates a new multi-layered fraud detector.
func NewFraudDetector() *FraudDetector {
	return &FraudDetector{
		isolationForest:   NewStatisticalIsolationForest(),
		sequenceAnalyzer:  NewTxSequenceAnalyzer(),
		sybilDetector:     NewSybilDetector(),
		ddosDetector:      NewDDoSDetector(100), // 100 TX/min threshold
		flashLoanDetector: NewFlashLoanDetector(),
		exploitDetector:   NewExploitDetector(),
	}
}

// DetectFraud runs all fraud detection layers on a transaction.
// The blockHeight parameter is used to generate deterministic investigation IDs
// that are safe for consensus. Callers should pass ctx.BlockHeight().
func (fd *FraudDetector) DetectFraud(_ context.Context, tx types.TransactionInfo, history []types.TransactionInfo, blockHeight int64) (*types.FraudResult, error) {
	results := make([]subResult, 0, 5)

	// Layer 1: Statistical Isolation Forest — general anomaly scoring
	isoScore := fd.isolationForest.Score(tx, history)
	if isoScore > 0.7 {
		results = append(results, subResult{
			threatType:  "unknown",
			threatLevel: scoreThreatLevel(isoScore),
			confidence:  isoScore,
			details:     fmt.Sprintf("isolation forest anomaly score: %.3f", isoScore),
		})
	}

	// Layer 2: Sequence analysis — detect unusual TX ordering patterns
	seqResult := fd.sequenceAnalyzer.Analyze(tx, history)
	if seqResult.score > 0.6 {
		results = append(results, subResult{
			threatType:  seqResult.threatType,
			threatLevel: scoreThreatLevel(seqResult.score),
			confidence:  seqResult.score,
			details:     seqResult.details,
		})
	}

	// Layer 3: Sybil detection — sudden spike in new addresses
	sybilResult := fd.sybilDetector.Check(tx, history)
	if sybilResult.confidence > 0.5 {
		results = append(results, subResult{
			threatType:  "sybil",
			threatLevel: scoreThreatLevel(sybilResult.confidence),
			confidence:  sybilResult.confidence,
			details:     sybilResult.details,
		})
	}

	// Layer 4: DDoS detection — unusual TX volume from specific sources
	ddosResult := fd.ddosDetector.Check(tx, history)
	if ddosResult.confidence > 0.5 {
		results = append(results, subResult{
			threatType:  "ddos",
			threatLevel: scoreThreatLevel(ddosResult.confidence),
			confidence:  ddosResult.confidence,
			details:     ddosResult.details,
		})
	}

	// Layer 5: Flash loan detection — specific TX sequence patterns
	flashResult := fd.flashLoanDetector.Check(tx, history)
	if flashResult.confidence > 0.5 {
		results = append(results, subResult{
			threatType:  "flash_loan",
			threatLevel: scoreThreatLevel(flashResult.confidence),
			confidence:  flashResult.confidence,
			details:     flashResult.details,
		})
	}

	// Layer 6: Exploit detection — abnormal gas consumption patterns
	exploitResult := fd.exploitDetector.Check(tx, history)
	if exploitResult.confidence > 0.5 {
		results = append(results, subResult{
			threatType:  "exploit",
			threatLevel: scoreThreatLevel(exploitResult.confidence),
			confidence:  exploitResult.confidence,
			details:     exploitResult.details,
		})
	}

	// Aggregate: take the highest-confidence detection
	if len(results) == 0 {
		return &types.FraudResult{
			ThreatLevel: "none",
			ThreatType:  "",
			Action:      "allow",
			Confidence:  0.95,
			Details:     "no fraud indicators detected",
		}, nil
	}

	// Find highest-confidence result
	best := results[0]
	for _, r := range results[1:] {
		if r.confidence > best.confidence {
			best = r
		}
	}

	// Determine response action per whitepaper:
	// 1. Alert: Emit event for validators and operators
	// 2. Rate Limit: Temporarily reduce TX acceptance
	// 3. Circuit Breaker: Pause specific contract executions
	// 4. Investigation: Generate detailed report
	action := determineAction(best.threatLevel, best.confidence)

	investigationID := ""
	if action != "allow" {
		hashSuffix := tx.TxHash
		if len(hashSuffix) > 8 {
			hashSuffix = hashSuffix[:8]
		}
		investigationID = fmt.Sprintf("INV-%d-%s", blockHeight, hashSuffix)
	}

	return &types.FraudResult{
		ThreatLevel:     best.threatLevel,
		ThreatType:      best.threatType,
		Action:          action,
		Confidence:      best.confidence,
		Details:         best.details,
		InvestigationID: investigationID,
	}, nil
}

type subResult struct {
	threatType  string
	threatLevel string
	confidence  float64
	details     string
}

// truncateAddr safely truncates an address to at most 12 characters for display.
func truncateAddr(addr string) string {
	if len(addr) > 12 {
		return addr[:12]
	}
	return addr
}

func scoreThreatLevel(score float64) string {
	switch {
	case score >= 0.9:
		return "critical"
	case score >= 0.7:
		return "high"
	case score >= 0.5:
		return "medium"
	case score >= 0.3:
		return "low"
	default:
		return "none"
	}
}

func determineAction(threatLevel string, confidence float64) string {
	switch threatLevel {
	case "critical":
		if confidence > 0.8 {
			return "circuit_break"
		}
		return "rate_limit"
	case "high":
		if confidence > 0.7 {
			return "rate_limit"
		}
		return "alert"
	case "medium":
		return "alert"
	default:
		return "allow"
	}
}

// ---- Statistical Isolation Forest ----

// StatisticalIsolationForest implements a simplified isolation forest using
// statistical distance measures. Full ML model integration is via the sidecar.
type StatisticalIsolationForest struct{}

func NewStatisticalIsolationForest() *StatisticalIsolationForest {
	return &StatisticalIsolationForest{}
}

// Score computes an isolation score for the transaction.
// Higher score = more anomalous (0.0 to 1.0).
func (f *StatisticalIsolationForest) Score(tx types.TransactionInfo, history []types.TransactionInfo) float64 {
	if len(history) < 5 {
		return 0.0 // Not enough data for meaningful scoring
	}

	// Feature 1: Amount deviation from population mean
	amountScore := f.amountDeviation(tx, history)

	// Feature 2: Gas usage deviation
	gasScore := f.gasDeviation(tx, history)

	// Feature 3: Sender frequency anomaly
	freqScore := f.senderFrequencyAnomaly(tx, history)

	// Combine features using max (isolation forest style — any single anomalous feature is suspicious)
	return math.Max(amountScore, math.Max(gasScore, freqScore))
}

func (f *StatisticalIsolationForest) amountDeviation(tx types.TransactionInfo, history []types.TransactionInfo) float64 {
	if tx.Amount == 0 {
		return 0.0
	}
	var sum, sumSq float64
	var count int
	for _, h := range history {
		if h.Amount > 0 {
			v := float64(h.Amount)
			sum += v
			sumSq += v * v
			count++
		}
	}
	if count < 3 {
		return 0.0
	}
	mean := sum / float64(count)
	variance := (sumSq / float64(count)) - (mean * mean)
	if variance <= 0 {
		return 0.0
	}
	stddev := math.Sqrt(variance)
	zScore := math.Abs(float64(tx.Amount)-mean) / stddev
	// Map z-score to 0-1 range: z=3 → 0.6, z=5 → 1.0
	return math.Min(zScore/5.0, 1.0)
}

func (f *StatisticalIsolationForest) gasDeviation(tx types.TransactionInfo, history []types.TransactionInfo) float64 {
	if tx.GasUsed == 0 {
		return 0.0
	}
	var sum, sumSq float64
	var count int
	for _, h := range history {
		if h.GasUsed > 0 {
			v := float64(h.GasUsed)
			sum += v
			sumSq += v * v
			count++
		}
	}
	if count < 3 {
		return 0.0
	}
	mean := sum / float64(count)
	variance := (sumSq / float64(count)) - (mean * mean)
	if variance <= 0 {
		return 0.0
	}
	stddev := math.Sqrt(variance)
	zScore := math.Abs(float64(tx.GasUsed)-mean) / stddev
	return math.Min(zScore/5.0, 1.0)
}

func (f *StatisticalIsolationForest) senderFrequencyAnomaly(tx types.TransactionInfo, history []types.TransactionInfo) float64 {
	// Count TXs from this sender in history
	senderCount := 0
	totalCount := len(history)
	for _, h := range history {
		if h.Sender == tx.Sender {
			senderCount++
		}
	}
	if totalCount == 0 {
		return 0.0
	}
	// Sender frequency as fraction of total
	freq := float64(senderCount) / float64(totalCount)
	// If sender accounts for >20% of recent TXs, that's suspicious
	if freq > 0.2 {
		return math.Min(freq*2.0, 1.0) // 50% → 1.0
	}
	return 0.0
}

// ---- TX Sequence Analyzer ----

// TxSequenceAnalyzer detects suspicious transaction ordering patterns.
type TxSequenceAnalyzer struct{}

func NewTxSequenceAnalyzer() *TxSequenceAnalyzer {
	return &TxSequenceAnalyzer{}
}

type sequenceResult struct {
	score      float64
	threatType string
	details    string
}

// Analyze checks for suspicious transaction sequences.
func (a *TxSequenceAnalyzer) Analyze(tx types.TransactionInfo, history []types.TransactionInfo) sequenceResult {
	// Pattern: rapid alternating send/receive (potential wash trading)
	if len(history) < 5 {
		return sequenceResult{score: 0}
	}

	// Count alternating patterns (sender→receiver then receiver→sender)
	alternatingCount := 0
	for _, h := range history {
		if h.Sender == tx.Receiver && h.Receiver == tx.Sender {
			alternatingCount++
		}
	}

	if alternatingCount > 3 {
		score := math.Min(float64(alternatingCount)/5.0, 1.0)
		return sequenceResult{
			score:      score,
			threatType: "wash_trading",
			details:    fmt.Sprintf("detected %d alternating transfers between %s and %s", alternatingCount, truncateAddr(tx.Sender), truncateAddr(tx.Receiver)),
		}
	}

	return sequenceResult{score: 0}
}

// ---- Sybil Detector ----

// SybilDetector detects sudden spikes in new unique addresses.
type SybilDetector struct {
	mu              sync.Mutex
	recentAddresses map[string]time.Time
}

func NewSybilDetector() *SybilDetector {
	return &SybilDetector{
		recentAddresses: make(map[string]time.Time),
	}
}

func (d *SybilDetector) Check(tx types.TransactionInfo, history []types.TransactionInfo) subResult {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Track unique senders in recent history
	uniqueSenders := make(map[string]bool)
	for _, h := range history {
		uniqueSenders[h.Sender] = true
	}

	// Check if sender is new (not in history)
	isNewSender := !uniqueSenders[tx.Sender]

	// Count new addresses in the recent window
	newAddressCount := 0
	for _, h := range history {
		if _, seen := d.recentAddresses[h.Sender]; !seen {
			newAddressCount++
		}
	}

	// Update tracking
	d.recentAddresses[tx.Sender] = time.Now()

	// Sybil signal: >30% of recent TXs are from new addresses
	if len(history) > 10 {
		newRatio := float64(newAddressCount) / float64(len(history))
		if newRatio > 0.3 && isNewSender {
			return subResult{
				confidence: math.Min(newRatio*2.0, 1.0),
				details:    fmt.Sprintf("%.0f%% of recent transactions from new addresses (potential Sybil)", newRatio*100),
			}
		}
	}

	return subResult{confidence: 0}
}

// ---- DDoS Detector ----

// DDoSDetector detects unusual transaction volume spikes from specific sources.
type DDoSDetector struct {
	maxTxPerMinute int
}

func NewDDoSDetector(maxTxPerMinute int) *DDoSDetector {
	return &DDoSDetector{maxTxPerMinute: maxTxPerMinute}
}

func (d *DDoSDetector) Check(tx types.TransactionInfo, history []types.TransactionInfo) subResult {
	// Count recent TXs from sender
	senderCount := 0
	for _, h := range history {
		if h.Sender == tx.Sender {
			senderCount++
		}
	}

	if senderCount > d.maxTxPerMinute {
		conf := math.Min(float64(senderCount)/float64(d.maxTxPerMinute*2), 1.0)
		return subResult{
			confidence: conf,
			details:    fmt.Sprintf("sender %s has %d recent transactions (limit: %d)", truncateAddr(tx.Sender), senderCount, d.maxTxPerMinute),
		}
	}

	return subResult{confidence: 0}
}

// ---- Flash Loan Detector ----

// FlashLoanDetector identifies specific TX sequence patterns characteristic of flash loan attacks:
// borrow → manipulate → profit → repay within a single block or short sequence.
type FlashLoanDetector struct{}

func NewFlashLoanDetector() *FlashLoanDetector {
	return &FlashLoanDetector{}
}

func (d *FlashLoanDetector) Check(tx types.TransactionInfo, history []types.TransactionInfo) subResult {
	// Pattern: large borrow followed by multiple small operations then large repay
	// Look for large amount variance within same sender's recent TXs
	var senderTxs []types.TransactionInfo
	for _, h := range history {
		if h.Sender == tx.Sender && h.Height == tx.Height { // Same block
			senderTxs = append(senderTxs, h)
		}
	}

	if len(senderTxs) < 3 {
		return subResult{confidence: 0}
	}

	// Check for large variance in amounts (flash loan pattern)
	var minAmt, maxAmt uint64
	minAmt = math.MaxUint64
	for _, t := range senderTxs {
		if t.Amount < minAmt {
			minAmt = t.Amount
		}
		if t.Amount > maxAmt {
			maxAmt = t.Amount
		}
	}

	if maxAmt > 0 && minAmt < maxAmt/10 { // 10x variance in same block
		conf := math.Min(float64(len(senderTxs))/10.0, 0.9)
		return subResult{
			confidence: conf,
			details:    fmt.Sprintf("flash loan pattern: %d txs in same block with amount range %d-%d", len(senderTxs), minAmt, maxAmt),
		}
	}

	return subResult{confidence: 0}
}

// ---- Exploit Detector ----

// ExploitDetector identifies transactions with abnormal gas consumption patterns
// that may indicate smart contract exploitation.
type ExploitDetector struct{}

func NewExploitDetector() *ExploitDetector {
	return &ExploitDetector{}
}

func (d *ExploitDetector) Check(tx types.TransactionInfo, history []types.TransactionInfo) subResult {
	if tx.GasUsed == 0 {
		return subResult{confidence: 0}
	}

	// Look for gas anomaly in contract calls
	if tx.TxType != "contract_call" && tx.TxType != "contract_deploy" {
		return subResult{confidence: 0}
	}

	// Compute average gas for similar TX types
	var sum float64
	var count int
	for _, h := range history {
		if h.TxType == tx.TxType && h.GasUsed > 0 {
			sum += float64(h.GasUsed)
			count++
		}
	}

	if count < 3 {
		return subResult{confidence: 0}
	}

	avgGas := sum / float64(count)
	ratio := float64(tx.GasUsed) / avgGas

	// If gas usage is >5x the average, flag as potential exploit
	if ratio > 5.0 {
		conf := math.Min((ratio-5.0)/10.0, 0.9)
		return subResult{
			confidence: conf,
			details:    fmt.Sprintf("gas usage %d is %.1fx average for %s operations", tx.GasUsed, ratio, tx.TxType),
		}
	}

	return subResult{confidence: 0}
}
