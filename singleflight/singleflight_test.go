package singleflight

import (
	"testing"
)

func TestDo(t *testing.T) {
	var g Group
	v, err := g.Do("key", func() (interface{}, error) {
		return "bar", nil
	})

	if v != "bar" || err != nil {
		t.Errorf("Do v = %v, error = %v", v, err)
	}
	
	c := make(chan string)
	var calls int
	fn := func() (interface{}, error) {
		calls++
		return <-c, nil
	}

	for i := 0; i < 10; i++ {
		go func() {
			g.Do("key", fn)
		}()
	}

	c <- "bar"

	if calls != 1 {
		t.Errorf("singleflight called %d times, want 1", calls)
	}
}
