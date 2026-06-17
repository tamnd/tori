// Package thread reconstructs conversations from a flat set of records. It is
// pure: tweets in, ordered threads out, no network and no clock, so it carries
// golden tests for the ordering rules (spec §14). A "thread" here is every
// captured tweet sharing a conversation id, ordered chronologically; a tweet
// alone in its conversation is a standalone post, represented as a one-tweet
// thread so the renderers treat both uniformly.
package thread

import (
	"sort"

	"github.com/tamnd/x-cli/x"
)

// Thread is one reconstructed conversation: a root tweet and every captured
// tweet in the same conversation, ordered oldest-first.
type Thread struct {
	RootID string
	Root   *x.Tweet
	Tweets []*x.Tweet
}

// Standalone reports whether the thread is a single post with no captured
// replies — the common case for a profile timeline.
func (t Thread) Standalone() bool { return len(t.Tweets) <= 1 }

// Assemble groups tweets into threads keyed by conversation id. Within a thread
// the tweets are ordered by id ascending (chronological); the root is the tweet
// whose id equals the conversation id when present, else the earliest tweet in
// the group. The returned threads are ordered by root id descending (newest
// first), the natural reverse-chronological order for a profile index. The order
// is a pure function of the input, so it is reproducible (TP5).
func Assemble(tweets []*x.Tweet) []Thread {
	groups := map[string][]*x.Tweet{}
	for _, t := range tweets {
		if t == nil || t.ID == "" {
			continue
		}
		conv := t.ConversationID
		if conv == "" {
			conv = t.ID
		}
		groups[conv] = append(groups[conv], t)
	}

	threads := make([]Thread, 0, len(groups))
	for conv, ts := range groups {
		sort.Slice(ts, func(i, j int) bool { return lessID(ts[i].ID, ts[j].ID) })
		root := ts[0]
		for _, t := range ts {
			if t.ID == conv {
				root = t
				break
			}
		}
		threads = append(threads, Thread{RootID: root.ID, Root: root, Tweets: ts})
	}

	sort.Slice(threads, func(i, j int) bool {
		// Newest root first; ties broken lexically for stability.
		return lessID(threads[j].RootID, threads[i].RootID)
	})
	return threads
}

// lessID compares numeric id strings by length then lexically — numeric order
// for non-negative snowflake ids without parsing or overflow.
func lessID(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}
