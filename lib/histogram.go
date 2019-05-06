package vegeta

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Buckets represents an Histogram's latency buckets.
type Buckets []time.Duration

// Histogram is a bucketed latency Histogram.
type Histogram struct {
	Buckets  Buckets
	Counts   []uint64
	Total    uint64
	Exponent uint
}

// makeBucket grows the bucket list so that r will fit.
func (h *Histogram) makeBucket(r *Result) int {
	for i := len(h.Buckets); ; i += 1 {
		h.Buckets = append(h.Buckets, time.Duration(h.Exponent)*h.Buckets[i-1])
		if r.Latency >= h.Buckets[i-1] && r.Latency < h.Buckets[i] {
			return i - 1
		}
	}
}

// getBucket returns the bucket index appropriate for r, adding buckets if needed.
func (h *Histogram) getBucket(r *Result) int {
	var i int

	// If matching bucket exists, return bucket
	for i = 0; i < len(h.Buckets)-1; i++ {
		if r.Latency >= h.Buckets[i] && r.Latency < h.Buckets[i+1] {
			return i // Never the last bucket
		}
	}

	// Else if automatic binning enabled, make bucket
	if h.Exponent > 0 {
		return h.makeBucket(r)
	}

	// Else, put in last bucket
	return i
}

// Add implements the Add method of the Report interface by finding the right
// Bucket for the given Result latency and increasing its count by one as well
// as the total count.
func (h *Histogram) Add(r *Result) {
	i := h.getBucket(r)

	// Extend h.Counts to match length of h.Buckets
	if len(h.Counts) < len(h.Buckets) {
		h.Counts = append(h.Counts, make([]uint64, len(h.Buckets)-len(h.Counts))...)
	}

	h.Total++
	h.Counts[i]++
}

// Close implements the Close method of the Report interface.
// If automatic logarithmic binning is enabled, it prunes unnecessary buckets.
// Otherwise it does nothing.
func (h *Histogram) Close() { // TODO actually call this
	var a, b int

	// Buckets are manually defined, don't touch
	if h.Exponent == 0 {
		return
	}

	// Find first non-empty bucket
	for a = 0; a < len(h.Counts); a++ {
		if h.Counts[a] > 0 {
			break
		}
	}

	// Find last non-empty bucket
	for b = len(h.Counts) - 1; b > a; b-- {
		if h.Counts[b] > 0 {
			break
		}
	}

	h.Counts = h.Counts[a:b+1]
	h.Buckets = h.Buckets[a:b+1]
}

// MarshalJSON returns a JSON encoding of the list of counts.
func (h *Histogram) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.Counts)
}

// Nth returns the nth bucket represented as a string.
func (bs Buckets) Nth(i int) (left, right string) {
	if i >= len(bs)-1 {
		return bs[i].String(), "+Inf"
	}
	return bs[i].String(), bs[i+1].String()
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (bs *Buckets) UnmarshalText(value []byte) error {
	if len(value) < 2 || value[0] != '[' || value[len(value)-1] != ']' {
		return fmt.Errorf("bad buckets: %s", value)
	}
	for _, v := range strings.Split(string(value[1:len(value)-1]), ",") {
		d, err := time.ParseDuration(strings.TrimSpace(v))
		if err != nil {
			return err
		}
		*bs = append(*bs, d)
	}
	if len(*bs) == 0 {
		return fmt.Errorf("bad buckets: %s", value)
	}
	return nil
}
