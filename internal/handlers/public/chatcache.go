package public

import (
	"context"
	"sync"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

// chatCacheSize is the hard cap on messages held in the in-process Connect cache. It
// comfortably covers a single poll delta (chatPollLimit=200) and the initial hydrate
// (chatRecentLimit=50), so an active poller is always served entirely from memory.
//
// Bounded RAM: each chatMessageVM is <=~1.5 KB (the body is capped at chatMaxBody=1000),
// so the whole buffer is worst-case ~256*1.5 KB ~= 0.4 MB - constant, independent of the
// total message history (which stays in MySQL) and of the number of visitors (one shared
// buffer, not per-client). Oldest messages are evicted as new ones arrive.
const chatCacheSize = 256

// chatHiddenCap bounds the ring of recently-hidden message ids the poll response carries so
// clients can remove those nodes live (see MarkHidden). Mirrors chatCacheSize; hides are
// rare, so this holds far more than any realistic burst.
const chatHiddenCap = 256

// ChatCache is a concurrency-safe, in-process cache of the most recent visible chat
// messages, already formatted as chatMessageVM. Its purpose is to serve the very frequent
// polling reads (every visitor, every 5s) from memory instead of querying MySQL on each
// request, which is the dominant chat load on a single-instance deployment.
//
// Writes still go to the DB; the cache is appended on a successful post and re-warmed
// lazily after a moderation change (see Invalidate). It intentionally holds no negative
// state: a cold or unwarmable cache reports a miss and the caller falls back to the DB, so
// correctness never depends on the cache being populated.
//
// Single-instance only: like the in-memory rate limiter, this cache is per-process and does
// not coordinate across instances (see the deploy notes in CLAUDE.md).
type ChatCache struct {
	q *sqlc.Queries // nil disables the cache; reads then always miss to the (also nil-safe) DB path

	mu     sync.RWMutex
	msgs   []chatMessageVM // oldest -> newest, len <= chatCacheSize
	full   bool            // true once older visible messages have been evicted (a deep cursor may miss)
	warmed bool            // false forces a DB re-warm on the next read
	hidden []uint64        // recently-hidden ids the poll response ships so clients can remove them live, len <= chatHiddenCap
}

// newChatCache builds an empty cache; it warms lazily on the first read.
func newChatCache(q *sqlc.Queries) *ChatCache { return &ChatCache{q: q} }

// ensureWarm loads the recent history from the DB the first time it is needed (and again
// after Invalidate). A load error leaves the cache unwarmed so reads keep missing to the DB
// rather than serving a wrong (empty) view.
func (c *ChatCache) ensureWarm(ctx context.Context) {
	c.mu.RLock()
	warmed := c.warmed
	c.mu.RUnlock()
	if warmed {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.warmed || c.q == nil {
		return
	}
	rows, err := c.q.ListRecentChatMessages(ctx, int32(chatCacheSize))
	if err != nil {
		return
	}
	msgs := make([]chatMessageVM, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- { // rows come newest-first; store oldest-first
		m := rows[i]
		msgs = append(msgs, chatMessageVM{
			ID: m.ID, UserID: m.ChatUserID, Name: m.AuthorName, Avatar: m.AuthorAvatar.String,
			IsAdmin: m.AuthorIsAdmin, Body: m.Body, Time: formatChatTime(m.CreatedAt),
		})
	}
	c.msgs = msgs
	// A full page back from the DB means there may be older visible messages we did not
	// load; fewer means we hold the entire history and can cover any cursor.
	c.full = len(rows) == chatCacheSize
	c.warmed = true
}

// Since returns the cached messages with id > sinceID, oldest->newest, and ok=true when the
// cache fully covers that range. It returns ok=false - the caller then falls back to the DB
// - when the cache could not be warmed or when sinceID predates the oldest cached message
// (a long-idle tab reconnecting), so no backlog is ever silently dropped.
func (c *ChatCache) Since(ctx context.Context, sinceID uint64) ([]chatMessageVM, bool) {
	c.ensureWarm(ctx)
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.warmed {
		return nil, false
	}
	if len(c.msgs) == 0 {
		return []chatMessageVM{}, true
	}
	// If older visible messages were evicted, a cursor at or below the message before the
	// oldest held one might be missing rows in (sinceID, oldest); defer to the DB.
	if c.full && sinceID+1 < c.msgs[0].ID {
		return nil, false
	}
	out := make([]chatMessageVM, 0, len(c.msgs))
	for _, m := range c.msgs {
		if m.ID > sinceID {
			out = append(out, m)
		}
	}
	return out, true
}

// Recent returns the last n cached messages, oldest->newest, matching the no-"since" poll
// shape. ok=false when the cache could not be warmed (caller falls back to the DB).
func (c *ChatCache) Recent(ctx context.Context, n int) ([]chatMessageVM, bool) {
	c.ensureWarm(ctx)
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.warmed {
		return nil, false
	}
	if n > len(c.msgs) {
		n = len(c.msgs)
	}
	out := make([]chatMessageVM, n)
	copy(out, c.msgs[len(c.msgs)-n:])
	return out, true
}

// Append adds a freshly posted message, evicting the oldest past the cap. A not-yet-warmed
// cache skips the append: the message is already in the DB and the next read warms from
// there, so it is never lost.
func (c *ChatCache) Append(m chatMessageVM) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.warmed {
		return
	}
	c.msgs = append(c.msgs, m)
	if len(c.msgs) > chatCacheSize {
		c.msgs = c.msgs[len(c.msgs)-chatCacheSize:]
		c.full = true
	}
}

// MarkHidden records a moderator-hidden message so the poll response tells clients to remove
// it live (see Hidden), and splices it out of the visible cache if present. It touches no DB
// and keeps the cache warm - hides are the common moderation action.
func (c *ChatCache) MarkHidden(id uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Drop from the visible slice so a fresh read no longer serves it (no re-warm needed).
	for i, m := range c.msgs {
		if m.ID == id {
			c.msgs = append(c.msgs[:i], c.msgs[i+1:]...)
			break
		}
	}
	// Record in the hidden ring (dedup, evict oldest past the cap).
	for _, h := range c.hidden {
		if h == id {
			return
		}
	}
	c.hidden = append(c.hidden, id)
	if len(c.hidden) > chatHiddenCap {
		c.hidden = c.hidden[len(c.hidden)-chatHiddenCap:]
	}
}

// MarkUnhidden reverses a hide: it removes the id from the hidden ring - essential, or a
// fresh client that just hydrated the restored message would be told to delete it - then
// drops the visible cache so the next read re-warms and reincludes the message for new loads.
// (Clients that already removed it live get it back on their next reload, not instantly.)
func (c *ChatCache) MarkUnhidden(id uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, h := range c.hidden {
		if h == id {
			c.hidden = append(c.hidden[:i], c.hidden[i+1:]...)
			break
		}
	}
	c.msgs = nil
	c.full = false
	c.warmed = false
}

// Hidden returns a copy of the current recently-hidden id set for the poll response.
func (c *ChatCache) Hidden() []uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.hidden) == 0 {
		return []uint64{}
	}
	out := make([]uint64, len(c.hidden))
	copy(out, c.hidden)
	return out
}
