package monitor

import (
	"sort"
	"sync"
	"time"

	"github.com/erigontech/erigon-lib/metrics"
)

var (
	// VALIDATOR METRICS

	// metricAttestHit is the number of attestations that hit for those validators we observe within current_epoch-2
	metricAttestHit = metrics.GetOrCreateCounter("validator_attestation_hit")
	// metricAttestMiss is the number of attestations that miss for those validators we observe within current_epoch-2
	metricAttestMiss = metrics.GetOrCreateCounter("validator_attestation_miss")
	// metricProposerHit is the number of proposals that hit for those validators we observe in previous slot
	metricProposerHit = metrics.GetOrCreateCounter("validator_proposal_hit")
	// metricProposerMiss is the number of proposals that miss for those validators we observe in previous slot
	metricProposerMiss = metrics.GetOrCreateCounter("validator_proposal_miss")

	// Block processing metrics
	fullBlockProcessingTime        = metrics.GetOrCreateGauge("full_block_processing_time")
	attestationBlockProcessingTime = metrics.GetOrCreateGauge("attestation_block_processing_time")
	batchVerificationThroughput    = metrics.GetOrCreateGauge("aggregation_per_signature")

	// Epoch processing metrics
	epochProcessingTime = metrics.GetOrCreateGauge("epoch_processing_time")

	// Network metrics
	gossipTopicsMetricCounterPrefix = "gossip_topics_seen"
	gossipMetricsMap                = sync.Map{}
	aggregateQuality50Per           = metrics.GetOrCreateGauge("aggregate_quality_50")
	aggregateQuality25Per           = metrics.GetOrCreateGauge("aggregate_quality_25")
	aggregateQuality75Per           = metrics.GetOrCreateGauge("aggregate_quality_75")
	aggregateQualityMin             = metrics.GetOrCreateGauge("aggregate_quality_min")
	aggregateQualityMax             = metrics.GetOrCreateGauge("aggregate_quality_max")

	// Beacon chain metrics
	committeeSize = metrics.GetOrCreateGauge("committee_size")
)

type batchVerificationThroughputMetric struct {
	totalVerified      uint64
	currentAverageSecs float64
	mu                 sync.Mutex
}

type aggregateQualityMetric struct {
	qualities []float64
	mu        sync.Mutex
}

func (a *aggregateQualityMetric) observe(participationCount int, totalCount int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	newPercentage := float64(participationCount) / float64(totalCount)
	a.qualities = append(a.qualities, newPercentage)
	if len(a.qualities) <= 40 {
		return
	}
	sort.Float64s(a.qualities)
	aggregateQuality50Per.Set(a.qualities[len(a.qualities)/2])
	aggregateQuality25Per.Set(a.qualities[len(a.qualities)/4])
	aggregateQuality75Per.Set(a.qualities[(len(a.qualities)*3)/4])
	aggregateQualityMin.Set(a.qualities[0])
	aggregateQualityMax.Set(a.qualities[len(a.qualities)-1])

	a.qualities = a.qualities[:0]

}

var (
	batchVerificationThroughputMetricStruct = &batchVerificationThroughputMetric{}
	aggregateQualityMetricStruct            = &aggregateQualityMetric{}
)

func (b *batchVerificationThroughputMetric) observe(t time.Duration, totalSigs int) float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	elapsedInMillisecs := float64(t.Microseconds()) / 1000
	if b.totalVerified == 0 {
		b.currentAverageSecs = elapsedInMillisecs
	} else {
		b.currentAverageSecs = (b.currentAverageSecs*float64(b.totalVerified) + elapsedInMillisecs) / float64(b.totalVerified+uint64(totalSigs))
	}
	b.totalVerified += uint64(totalSigs)
	ret := b.currentAverageSecs
	if b.totalVerified > 1000 {
		b.currentAverageSecs = 0
		b.totalVerified = 0
	}
	return ret
}

func microToMilli(micros int64) float64 {
	return float64(micros) / 1000
}

// ObserveEpochProcessingTime sets last epoch processing time
func ObserveEpochProcessingTime(startTime time.Time) {
	epochProcessingTime.Set(float64(time.Since(startTime).Microseconds()))
}

// ObserveAttestHit increments the attestation hit metric
func ObserveAttestationBlockProcessingTime(startTime time.Time) {
	attestationBlockProcessingTime.Set(microToMilli(time.Since(startTime).Microseconds()))
}

// ObserveFullBlockProcessingTime increments the full block processing time metric
func ObserveFullBlockProcessingTime(startTime time.Time) {
	fullBlockProcessingTime.Set(microToMilli(time.Since(startTime).Microseconds()))
}

// ObserveBatchVerificationThroughput increments the batch verification throughput metric
func ObserveBatchVerificationThroughput(d time.Duration, totalSigs int) {
	batchVerificationThroughput.Set(batchVerificationThroughputMetricStruct.observe(d, totalSigs))
}

// ObserveGossipTopicSeen increments the gossip topic seen metric
func ObserveGossipTopicSeen(topic string, l int) {
	var metric metrics.Counter
	metricI, ok := gossipMetricsMap.LoadOrStore(topic, metrics.GetOrCreateCounter(gossipTopicsMetricCounterPrefix+"_"+topic))
	if ok {
		metric = metricI.(metrics.Counter)
	} else {
		metric = metrics.GetOrCreateCounter(gossipTopicsMetricCounterPrefix + "_" + topic)
		gossipMetricsMap.Store(topic, metric)
	}
	metric.Add(float64(l))
}

func ObserveAggregateQuality(participationCount int, totalCount int) {
	aggregateQualityMetricStruct.observe(participationCount, totalCount)
}

func ObserveCommitteeSize(size float64) {
	committeeSize.Set(size)
}
