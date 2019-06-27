package main

import (
	"errors"
	"reflect"
	"testing"
)

var normalizeInstanceScenarios = []struct {
	input       map[string]int64
	expected    map[string]float32
	expectedErr error
}{
	{
		map[string]int64{
			"c5.xlarge":  40,
			"c5.2xlarge": 50,
			"m4.4xlarge": 10,
			"m4.large":   100,
		},
		map[string]float32{
			"c5": 280,
			"m4": 180,
		},
		nil,
	},
	{
		map[string]int64{
			"c5":    10,
			"large": 20,
		},
		map[string]float32{},
		errors.New(BadInstancesInput),
	},
}

// test correct calculations
func TestNormalizeInstances(t *testing.T) {

	for _, scenario := range normalizeInstanceScenarios {
		res, err := normalizeInstances(scenario.input)

		if !reflect.DeepEqual(res, scenario.expected) {
			t.Errorf("Expected normalizeInstances to be:\n`%v`\nGot:\n`%v`", scenario.expected, res)
		}

		if !reflect.DeepEqual(err, scenario.expectedErr) {
			t.Errorf("Did not get expected error. Expected:\n`%v`\nGot:\n`%v`", scenario.expectedErr, err)
		}
	}
}
