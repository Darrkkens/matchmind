package service

import "testing"

func TestSimulationRunsFromQuestion(t *testing.T) {
	cases := map[string]int{
		"Simule o próximo jogo":                      DefaultSimulationRuns,
		"Explique a simulação com 10.000 simulações": 10000,
		"rode 200 vezes":                             200,
		"simule 99999 vezes":                         DefaultSimulationRuns,
	}
	for q, want := range cases {
		if got := simulationRuns(q); got != want {
			t.Errorf("%q: got %d, want %d", q, got, want)
		}
	}
}
