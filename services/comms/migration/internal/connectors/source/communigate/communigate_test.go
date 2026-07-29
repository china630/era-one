package communigate

import "testing"

func TestFolderMappingGolden(t *testing.T) {
	want := GoldenMappings()
	for cg, target := range want {
		if got := MapFolder(cg); got != target {
			t.Fatalf("MapFolder(%q)=%q want %q", cg, got, target)
		}
	}
}
