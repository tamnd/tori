package repo

import (
	"testing"
	"time"

	"github.com/tamnd/x-cli/x"
)

func TestStoreTweetRoundtrip(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	in := &x.Tweet{ID: "200", Text: "hello", CreatedAt: time.Unix(1700000000, 0).UTC()}
	if err := st.WriteTweet(in, nil); err != nil {
		t.Fatal(err)
	}
	if !st.HasTweet("200") {
		t.Fatal("HasTweet should report a written tweet")
	}
	got, err := st.LoadTweet("200")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "200" || got.Text != "hello" {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
}

func TestLoadTweetsSortedByID(t *testing.T) {
	dir := t.TempDir()
	st, _ := Open(dir)
	for _, id := range []string{"205", "100", "200"} {
		if err := st.WriteTweet(&x.Tweet{ID: id}, nil); err != nil {
			t.Fatal(err)
		}
	}
	all, err := st.LoadTweets()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"100", "200", "205"}
	if len(all) != len(want) {
		t.Fatalf("want %d tweets, got %d", len(want), len(all))
	}
	for i, w := range want {
		if all[i].ID != w {
			t.Errorf("tweet %d = %s, want %s", i, all[i].ID, w)
		}
	}
}

func TestManifestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	m := NewManifest(TargetRef{Kind: "profile", Ref: "jack"}, "v1.2.3")
	m.Tweets = 3
	m.AddCapture("2025-01-02T03:04:05Z", 3, "guest")
	if err := m.Save(dir); err != nil {
		t.Fatal(err)
	}
	got, ok, err := LoadManifest(dir)
	if err != nil || !ok {
		t.Fatalf("load: ok=%v err=%v", ok, err)
	}
	if got.Target.Ref != "jack" || got.Tweets != 3 || got.Schema != SchemaVersion {
		t.Fatalf("manifest mismatch: %+v", got)
	}
	if len(got.Captures) != 1 || got.Captures[0].Tier != "guest" {
		t.Fatalf("capture not persisted: %+v", got.Captures)
	}
}

func TestLoadManifestMissing(t *testing.T) {
	_, ok, err := LoadManifest(t.TempDir())
	if err != nil {
		t.Fatalf("missing manifest should not error, got %v", err)
	}
	if ok {
		t.Fatal("missing manifest should report ok=false")
	}
}
