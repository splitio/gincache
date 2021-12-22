package gincache

import (
	"fmt"
	"runtime"
	"strconv"
	"testing"
)

func TestCacheEvictBySurrogate(t *testing.T) {
	c := newMtxCache(100)
	c.trySet("entry1", []string{"s1", "s2"}, 200, []byte(`{"prop1": "val1"}`), map[string]string{"h1": "v1"}, false)
	s, b, h := c.get("entry1")
	if s != 200 {
		t.Error("wrong status")
	}

	if string(b) != `{"prop1": "val1"}` {
		t.Error("wrong body")
	}

	if len(h) != 1 || h["h1"] != "v1" {
		t.Error("wrong headers")
	}

	c.trySet("entry2", []string{"s1"}, 200, []byte(`{"prop1": "val1"}`), map[string]string{"h1": "v2"}, false)
	c.trySet("entry3", []string{"s3"}, 200, []byte(`{"prop1": "val1"}`), map[string]string{"h1": "v3"}, false)
	c.trySet("entry4", []string{"s2"}, 200, []byte(`{"prop1": "val1"}`), map[string]string{"h1": "v4"}, false)

	e2s, _, _ := c.get("entry2")
	e3s, _, _ := c.get("entry3")
	e4s, _, _ := c.get("entry4")
	e5s, _, _ := c.get("entry5")
	if e2s != 200 || e3s != 200 || e4s != 200 || e5s != 0 {
		t.Error("incorrect status codes: ", e2s, e3s, e4s, e5s)
	}

	c.evictBySurrogate("s1")

	e1s, _, _ := c.get("entry1")
	e2s, _, _ = c.get("entry2")
	e3s, _, _ = c.get("entry3")
	e4s, _, _ = c.get("entry4")

	if e1s != 0 || e2s != 0 {
		t.Error("entries should not be cached")
		fmt.Printf("%+v\n", c.surrogates)
	}

	if e3s != 200 || e4s != 200 {
		t.Error("entries should be cached")
	}

	if len(c.surrogates) != 2 {
		t.Error("there should be 2 surrobates only")
	}

	if _, ok := c.surrogates["s1"]; ok {
		t.Error("s1 should NOT be in the surrogates list")
	}

	if _, ok := c.surrogates["s2"]; !ok {
		t.Error("s2 should be in the surrogates list")
	}

	if _, ok := c.surrogates["s3"]; !ok {
		t.Error("s3 should be in the surrogates list")
	}

	c.evictBySurrogate("s2")
	e1s, _, _ = c.get("entry1")
	e2s, _, _ = c.get("entry2")
	e3s, _, _ = c.get("entry3")
	e4s, _, _ = c.get("entry4")
	if e4s != 0 {
		t.Error("entry4 should have been evicted")
	}

	if len(c.surrogates) != 1 {
		t.Error("there should be 1 surrogate only")
	}
}

func TestCacheSizeBound(t *testing.T) {
	c := newMtxCache(100)
	for idx := 0; idx < 500; idx++ {
		c.trySet("entry"+strconv.Itoa(idx), []string{"s1", "s2"}, 200, []byte(`{"prop1": "val1"}`), map[string]string{"h1": "v1"}, false)
	}
	if len(c.data) != 100 {
		t.Error("only 100 items should be stored.")
	}
}

func TestRemoveEntryWorksProperly(t *testing.T) {
	c := newMtxCache(100)
	c.trySet("entry1", []string{"s1", "s2"}, 200, []byte(`{"prop1": "val1"}`), map[string]string{"h1": "v1"}, false)
	c.trySet("entry2", []string{"s1"}, 200, []byte(`{"prop1": "val1"}`), map[string]string{"h1": "v1"}, false)
	c.trySet("entry3", []string{"s2"}, 200, []byte(`{"prop1": "val1"}`), map[string]string{"h1": "v1"}, false)

	// validate initial status
	if !cacheContains(c, "entry1") || !cacheContains(c, "entry2") || !cacheContains(c, "entry3") {
		t.Error("cache should contain entry1, entry2, entry3")
	}

	if !surrogateReferences(c, "s1", "entry1") || !surrogateReferences(c, "s2", "entry1") {
		t.Error("s1 and s2 should reference entry1")
	}
	if !surrogateReferences(c, "s1", "entry2") {
		t.Error("s1 should reference entry2")
	}
	if !surrogateReferences(c, "s2", "entry3") {
		t.Error("s2 should reference entry3")
	}

	c.evictBySurrogate("s1")
	if cacheContains(c, "entry1") || cacheContains(c, "entry2") {
		t.Error("cache should no longer contain entry1 or entry2")
	}

	if surrogateReferences(c, "s1", "entry1") || surrogateReferences(c, "s1", "entry2") {
		t.Error("s1 should not referentce entry1 nor entry2")
	}

	if surrogateExists(c, "s1") {
		t.Error("surrogate s1 should no longer exist")
	}

	c.evictBySurrogate("s2")
	if cacheContains(c, "entry3") {
		t.Error("cache should no longer contain entry3")
	}

	if surrogateReferences(c, "s2", "entry3") {
		t.Error("s2 should not referentce entry3")
	}
	if surrogateExists(c, "s2") {
		t.Error("surrogate s2 should no longer exist")
	}

}

func TestMemoryGrowth(t *testing.T) {
	var m runtime.MemStats
	// Start clean:
	c := newMtxCache(1000000)

	// two passes of GC are needed to remove tainted allocs and start clean
	runtime.GC()
	runtime.GC()

	// get initial allocated bytes
	runtime.ReadMemStats(&m)
	initial := int64(m.Alloc)

	// we use surrogate keys s0-s9 to divide the cache space in 10 ~equal segments
	makeSurrogates := func(i int) []string { return []string{fmt.Sprintf("s%d", i%10)} }

	// populate the cache
	for idx := 0; idx < 100000; idx++ {
		c.trySet("entry"+strconv.Itoa(idx), makeSurrogates(idx), 200, []byte(`{"prop1": "val1"}`), map[string]string{"h1": "v1"}, false)
	}

	// get reading with full cache
	runtime.ReadMemStats(&m)
	afterInitialPopulation := int64(m.Alloc)

	c.evictBySurrogate("s0")

	// two cycles of GC to free memory associated to entries referenced by surrogate s0
	runtime.GC()
	runtime.GC()

	runtime.ReadMemStats(&m)
	afterEviction := int64(m.Alloc)

	// first we want to get an estimate on how much memory is used by the full cache
	fullSize := int64(afterInitialPopulation - initial)
	sizeWithoutS0 := int64(afterEviction - initial)

	if sizeWithoutS0 >= fullSize {
		t.Errorf("data should be smaller after eviction. Full: %d, after purge: %d", fullSize, sizeWithoutS0)
	}

	// now, the "data" in the cache should have been reduced by 10%, but it's likely that memory associated to map
	// structure wasn't freed (it only reallocates to meet higher requirements AFAIK), so in practice it's likely that the
	// decrease is a little smaller than the 10% expected
	expected := int64(float64(fullSize) * 0.9) // real size - 10%

	// we want at most a 10% drift between the expected and actual memory usage after eviction
	tolerance := 0.1
	delta := intAbs(expected - int64(sizeWithoutS0))

	if diff := diffPercent(float64(expected), float64(delta)); diff > tolerance {
		t.Errorf("delta requirements not met: (%f > %f).\n all values: %d, afterPurge: %d, expected: %d, delta: %d, tolerance: %f",
			diff, tolerance,
			fullSize, sizeWithoutS0, expected, delta, tolerance)
		t.Errorf("for debug: initial: %d, after populate: %d, after eviction: %d",
			initial, afterInitialPopulation, afterEviction)
	}

	// Now we will expire 4 more surogates and validate that we're not far from a 50% of the original size
	c.evictBySurrogate("s1")
	c.evictBySurrogate("s2")
	c.evictBySurrogate("s3")
	c.evictBySurrogate("s4")
	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&m)
	after5Evictions := int64(m.Alloc)
	sizeWithoutS0ToS4 := int64(after5Evictions - initial)
	expected = int64(float64(fullSize) * 0.5)
	delta = intAbs(expected - int64(sizeWithoutS0ToS4))

	if diff := diffPercent(float64(expected), float64(delta)); diff > tolerance {
		t.Errorf("delta requirements not met: (%f > %f).\n all values: %d, afterPurge: %d, expected: %d, delta: %d, tolerance: %f",
			diff, tolerance,
			fullSize, sizeWithoutS0ToS4, expected, delta, tolerance)
	}

	// this call is just for referencing `c`. Otherwise the GC would kill it after the last usage,
	// and memory metrics would not make sense
	c.get("nonexistantkey")

}

func diffPercent(n1 float64, n2 float64) float64 {
	if n1 == 0 || n2 == 0 { // avoid possible division by 0
		return 0
	}

	if n1 < n2 {
		return n1 / n2
	}

	return n2 / n1
}

func intAbs(abs int64) int64 {
	if abs > 0 {
		return abs
	}
	return -abs
}

func surrogateReferences(c *mtxCache, surrogate string, key string) bool {
	s, ok := c.surrogates[surrogate]
	if !ok {
		return false
	}

	_, ok = s[key]
	return ok
}

func cacheContains(c *mtxCache, key string) bool {
	_, ok := c.data[key]
	return ok
}

func surrogateExists(c *mtxCache, surrogate string) bool {
	_, ok := c.surrogates[surrogate]
	return ok
}
