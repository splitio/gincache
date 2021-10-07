package gincache

import (
	"fmt"
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
