package doppel

import (
	"math"
	"math/rand/v2"
	"time"
)

const (
	StatsBisectTimes = 70
	StatsLowK        = 0.01
	StatsHighK       = 10.0

	// do not calculate statistics if we have < than this number of durations
	MinDurationsToCalculate = 100

	// these values are taken from ok.ru. measured from moscow site.
	StatsDefaultK      = 0.37846373895785335
	StatsDefaultLambda = 1.73177086015485
)

// Stats is responsible for generating values that are distributed according
// to some statistical distribution.
//
// It follows several ideas:
//  1. Based on nginx and Cloudflare behaviour, even if server is eager
//     to send a lot, they all start with small TLS packets that are
//     approximately MTU-sized. After
//  2. After ~40 TLS records, server considers TCP session as somewhat solid
//     and reliable and ramps up to 4096.
//  3. After ~20 TLS records more it jumps to the max 16384 bytes and keep
//     this size as long as it can
//  4. If there is no any byte within a connection for a longer time period,
//     this counter resets.
//
// This is called Dynamic TLS Record Sizing
//   - https://blog.cloudflare.com/optimizing-tls-over-tcp-to-reduce-latency/
//   - https://community.f5.com/kb/technicalarticles/boosting-tls-performance-with-dynamic-record-sizing-on-big-ip/280798
//   - https://www.igvita.com/2013/10/24/optimizing-tls-record-size-and-buffering-latency/
//
// And this optimized for the very first byte, so web browsers could start to
// render as early as possible, showing user some preliminary results, optimizing
// for perceived latency.
//
// Since this is very typical for the website, we also aim for that.
//
// Another important idea is how delays between TLS packets are distributed.
// In case of sending huge heavy content with max sized record, delays have
// lognormal distribution. But a nature of a typical website shows that
// it eagers to deliver as fast as it can in a few very first records and
// could possibly slow down later.
//
// This is perfectly described by Weibull distribution:
//   - https://en.wikipedia.org/wiki/Weibull_distribution
//   - https://ieeexplore.ieee.org/document/6662948
//   - https://www.researchgate.net/publication/224621285_Traffic_modelling_and_cost_optimization_for_transmitting_traffic_messages_over_a_hybrid_broadcast_and_cellular_network
//   - https://ir.uitm.edu.my/id/eprint/105386/1/105386.pdf
//
// In other word, a combination of Dynamic TLS Record Sizing hints us for
// Weibull distribution.
type Stats struct {
	sizeLastRequested time.Time
	sizeCounter       int

	// https://en.wikipedia.org/wiki/Shape_parameter
	k float64
	// https://en.wikipedia.org/wiki/Scale_parameter
	lambda float64
}

func (d *Stats) Delay() time.Duration {
	// u ∈ (0, 1], avoids ln(0)
	u := 1.0 - rand.Float64()

	// X = λ·(-ln U)^(1/k)
	generated := d.lambda * math.Pow(-math.Log(u), 1.0/d.k)

	// generated is in milliseconds
	return time.Duration(generated * float64(time.Millisecond))
}

func (d *Stats) Size() int {
	if time.Since(d.sizeLastRequested) > TLSRecordSizeResetAfter {
		d.sizeCounter = 0
	}

	d.sizeLastRequested = time.Now()
	d.sizeCounter++

	switch {
	case d.sizeCounter <= TLSCounterAccelAfter:
		return TLSRecordSizeStart
	case d.sizeCounter <= TLSCounterMaxAfter:
		return TLSRecordSizeAccel
	}

	return TLSRecordSizeMax
}

func NewStats(durations []time.Duration) *Stats {
	n := float64(len(durations))

	// in milliseconds
	durFloats := make([]float64, len(durations))
	for i, v := range durations {
		durFloats[i] = float64(v.Microseconds()) / 1000.0
	}

	// The bisection solves the standard Weibull MLE equation for shape
	// parameter k. There is no any good formula for doing that so we
	// approximate it by several bisections. The number of operations
	// is statically defined by a constant.

	sumLog := 0.0
	for _, v := range durFloats {
		sumLog += math.Log(v)
	}

	lowK := StatsLowK
	highK := StatsHighK

	for range StatsBisectTimes {
		midK := (lowK + highK) / 2.0
		sumXK := 0.0
		sumXKLog := 0.0

		for _, v := range durFloats {
			xk := math.Pow(v, midK)
			sumXK += xk
			sumXKLog += xk * math.Log(v)
		}

		if (1.0/midK)+(sumLog/n)-(sumXKLog/sumXK) > 0 {
			lowK = midK
		} else {
			highK = midK
		}
	}

	k := (lowK + highK) / 2

	sumXK := 0.0
	for _, v := range durFloats {
		sumXK += math.Pow(v, k)
	}

	// λ = (Σxᵢᵏ / n)^(1/k)
	lambda := math.Pow(sumXK/n, 1.0/k)

	return &Stats{
		k:      k,
		lambda: lambda,
	}
}
