package thread

import (
	"testing"

	"github.com/tamnd/x-cli/x"
)

func tw(id, conv string) *x.Tweet { return &x.Tweet{ID: id, ConversationID: conv} }

func TestAssembleGroupsByConversation(t *testing.T) {
	in := []*x.Tweet{
		tw("100", "100"), // standalone
		tw("201", "200"), // reply in conversation 200
		tw("200", "200"), // root of conversation 200
		tw("300", ""),    // no conversation id -> own root
	}
	got := Assemble(in)
	if len(got) != 3 {
		t.Fatalf("want 3 threads, got %d", len(got))
	}
	// Newest root first: 300, 200, 100.
	wantRoots := []string{"300", "200", "100"}
	for i, w := range wantRoots {
		if got[i].RootID != w {
			t.Errorf("thread %d root = %s, want %s", i, got[i].RootID, w)
		}
	}
}

func TestAssembleRootIsConversationID(t *testing.T) {
	got := Assemble([]*x.Tweet{tw("205", "200"), tw("200", "200"), tw("203", "200")})
	if len(got) != 1 {
		t.Fatalf("want 1 thread, got %d", len(got))
	}
	th := got[0]
	if th.RootID != "200" {
		t.Errorf("root = %s, want 200 (the conversation id)", th.RootID)
	}
	if th.Standalone() {
		t.Error("a 3-tweet conversation is not standalone")
	}
	// Tweets ordered oldest-first by id.
	want := []string{"200", "203", "205"}
	for i, w := range want {
		if th.Tweets[i].ID != w {
			t.Errorf("tweet %d = %s, want %s", i, th.Tweets[i].ID, w)
		}
	}
}

func TestAssembleStandalone(t *testing.T) {
	got := Assemble([]*x.Tweet{tw("100", "100")})
	if len(got) != 1 || !got[0].Standalone() {
		t.Fatalf("a single tweet should be one standalone thread, got %+v", got)
	}
}

// Assemble is order-independent: shuffled input yields the same threads.
func TestAssembleDeterministic(t *testing.T) {
	a := Assemble([]*x.Tweet{tw("200", "200"), tw("201", "200"), tw("100", "100")})
	b := Assemble([]*x.Tweet{tw("100", "100"), tw("201", "200"), tw("200", "200")})
	if len(a) != len(b) {
		t.Fatalf("len mismatch %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].RootID != b[i].RootID {
			t.Errorf("thread %d root %s vs %s", i, a[i].RootID, b[i].RootID)
		}
	}
}
