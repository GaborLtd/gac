package diff

import "testing"

func TestLimitByLines(t *testing.T) {
	got := Limit("a\nb\nc\n", 100, 2)
	if got.Text != "a\nb\n" || !got.Truncated {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestLimitByBytesWithoutBreakingUTF8(t *testing.T) {
	got := Limit("中文\n第二行\n", len("中文\n"), 100)
	if got.Text != "中文\n" || !got.Truncated {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestLimitUsesFirstReachedLimit(t *testing.T) {
	got := Limit("one\ntwo\n", 3, 10)
	if got.Text != "" || !got.Truncated {
		t.Fatalf("unexpected result: %#v", got)
	}
}
