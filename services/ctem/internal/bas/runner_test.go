package bas

import (
	"context"
	"testing"
)

func TestSimulateLateralNil(t *testing.T) {
	var r Runner
	if err := r.SimulateLateral(context.Background(), "10.0.0.1"); err != nil {
		t.Fatal(err)
	}
}
