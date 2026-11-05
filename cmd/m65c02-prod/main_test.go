package main

import "testing"

func TestProductionCounter(t *testing.T) {
	if err := demo(1000); err != nil {
		t.Fatal(err)
	}
}

func TestProductionExtendedLoop(t *testing.T) {
	if err := demoLoop(1000, false); err != nil {
		t.Fatal(err)
	}
}
