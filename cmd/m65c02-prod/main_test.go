package main

import "testing"

func TestProductionCounter(t *testing.T) {
	if err := demo(1000); err != nil {
		t.Fatal(err)
	}
}
