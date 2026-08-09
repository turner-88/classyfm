package admin

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/classyfm/classyfm/internal/db/sqlc"
	"github.com/classyfm/classyfm/internal/schedule"
)

// chatEnabledKey is the settings row that gates public chat posting site-wide. Absent (or
// anything other than "0") means enabled — chat is on by default, an admin has to turn it
// off. The public post handlers read the same key so the kill switch takes effect there.
const chatEnabledKey = "chat_enabled"

// chatEnabled reports whether public chat posting is currently allowed. A missing row or a
// read error is treated as enabled (fail-open: a settings glitch must not silently mute the
// whole chat).
func (h *Handler) chatEnabled(r *http.Request) bool {
	if h.q == nil {
		return true
	}
	s, err := h.q.GetSetting(r.Context(), chatEnabledKey)
	if err != nil {
		return true
	}
	return s.V != "0"
}

// formatChatTime renders a message timestamp as clock time in the station's timezone,
// matching the public widget's formatting so admins see the same clock the chatters do.
func formatChatTime(t time.Time) string { return t.In(schedule.Loc).Format("15:04") }

// chatMessageVM is one rendered chat message for the admin moderation surfaces (the static
// table and the live JSON feed share it). It carries is_deleted so the row can be marked
// and offered an un-hide, and the author's ban state so the ban/unban control is correct.
type chatMessageVM struct {
	ID        uint64 `json:"id"`
	UserID    uint64 `json:"user_id"`
	Name      string `json:"name"`
	Avatar    string `json:"avatar"`
	IsAdmin   bool   `json:"is_admin"`
	IsBanned  bool   `json:"is_banned"`
	Body      string `json:"body"`
	Time      string `json:"time"`
	IsDeleted bool   `json:"is_deleted"`
}

type chatMessagesListData struct {
	Base        baseData
	Messages    []chatMessageVM
	Pagination  pagination
	ChatEnabled bool
	LastID      uint64 // highest message id on the page, the live poll's starting cursor
}

// ChatMessagesList renders the chat moderation table: every message (including hidden ones),
// newest first, searchable by body or author name. The page also carries the global chat
// on/off state (rendered as a kill-switch toggle) and a Static/Live view toggle.
func (h *Handler) ChatMessagesList(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	search, pattern := searchPattern(r)
	sort, dir := parseSort(r, "created_at", "desc", "created_at")

	total, err := h.q.CountChatMessagesAdmin(r.Context(), sqlc.CountChatMessagesAdminParams{Search: pattern})
	if err != nil {
		http.Error(w, "failed to load chat messages", http.StatusInternalServerError)
		return
	}
	pg := paginate(r, total, "/admin/chat", url.Values{"q": {search}, "sort": {sort}, "dir": {dir}})
	rows, err := h.q.ListChatMessagesAdmin(r.Context(), sqlc.ListChatMessagesAdminParams{
		Search: pattern, Sort: sort, Dir: dir, Limit: adminPageSize, Offset: pg.Offset(),
	})
	if err != nil {
		http.Error(w, "failed to load chat messages", http.StatusInternalServerError)
		return
	}

	messages := make([]chatMessageVM, 0, len(rows))
	var lastID uint64
	for _, m := range rows {
		messages = append(messages, chatMessageVM{
			ID: m.ID, UserID: m.ChatUserID, Name: m.AuthorName, Avatar: m.AuthorAvatar.String,
			IsAdmin: m.AuthorIsAdmin, IsBanned: m.AuthorIsBanned, Body: m.Body,
			Time: formatChatTime(m.CreatedAt), IsDeleted: m.IsDeleted,
		})
		if m.ID > lastID {
			lastID = m.ID
		}
	}

	h.r.Page(w, http.StatusOK, "admin/chat_messages_list", chatMessagesListData{
		Base:        h.base(r, "Chat", "chat"),
		Messages:    messages,
		Pagination:  pg,
		ChatEnabled: h.chatEnabled(r),
		LastID:      lastID,
	})
}

// chatLiveLimit caps a single live-poll delta so a long-idle tab reconnecting can't pull an
// unbounded backlog in one request (mirrors the public chatPollLimit).
const chatLiveLimit = 200

// ChatMessagesJSON is the live-mode polling endpoint (?since=<id> returns newer messages,
// including hidden ones). It is the admin-only counterpart of the public ConnectMessagesJSON;
// route auth already restricts it to logged-in admins.
func (h *Handler) ChatMessagesJSON(w http.ResponseWriter, r *http.Request) {
	out := []chatMessageVM{}
	if h.q != nil {
		since, _ := strconv.ParseUint(r.URL.Query().Get("since"), 10, 64)
		rows, _ := h.q.ListChatMessagesAdminSince(r.Context(), sqlc.ListChatMessagesAdminSinceParams{
			ID: since, Limit: chatLiveLimit,
		})
		for _, m := range rows {
			out = append(out, chatMessageVM{
				ID: m.ID, UserID: m.ChatUserID, Name: m.AuthorName, Avatar: m.AuthorAvatar.String,
				IsAdmin: m.AuthorIsAdmin, IsBanned: m.AuthorIsBanned, Body: m.Body,
				Time: formatChatTime(m.CreatedAt), IsDeleted: m.IsDeleted,
			})
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(map[string]any{"messages": out})
}

// ChatMessageToggleHide flips one message's soft-delete state: hide a visible message, or
// un-hide a previously moderated one.
func (h *Handler) ChatMessageToggleHide(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	m, err := h.q.GetChatMessage(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if m.IsDeleted {
		err = h.q.UnhideChatMessage(r.Context(), id)
	} else {
		err = h.q.SoftDeleteChatMessage(r.Context(), id)
	}
	if err != nil {
		http.Error(w, "failed to update message", http.StatusInternalServerError)
		return
	}
	// Keep the public poll cache in step: m.IsDeleted is the pre-toggle state, so a message
	// that WAS deleted has just been restored (un-hide), otherwise it was just hidden. A hide
	// also makes the poll response tell live clients to remove it.
	if h.chatCache != nil {
		if m.IsDeleted {
			h.chatCache.MarkUnhidden(id)
		} else {
			h.chatCache.MarkHidden(id)
		}
	}
	action, detail := "hide", "Hid chat message"
	if m.IsDeleted {
		action, detail = "unhide", "Un-hid chat message"
	}
	h.audit(r, action, "chat_message", &id, detail)
	http.Redirect(w, r, chatReturnDest(r, "/admin/chat"), http.StatusSeeOther)
}

type chatUsersListData struct {
	Base        baseData
	Users       []sqlc.ChatUser
	Pagination  pagination
	ChatEnabled bool
}

// ChatUsersList renders the chat users table (name/email, admin badge, ban state) with a
// ban/unban toggle per row.
func (h *Handler) ChatUsersList(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	search, pattern := searchPattern(r)
	total, err := h.q.CountChatUsers(r.Context(), sqlc.CountChatUsersParams{Search: pattern})
	if err != nil {
		http.Error(w, "failed to load chat users", http.StatusInternalServerError)
		return
	}
	pg := paginate(r, total, "/admin/chat/users", url.Values{"q": {search}})
	users, err := h.q.ListChatUsers(r.Context(), sqlc.ListChatUsersParams{
		Search: pattern, Limit: adminPageSize, Offset: pg.Offset(),
	})
	if err != nil {
		http.Error(w, "failed to load chat users", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/chat_users_list", chatUsersListData{
		Base:        h.base(r, "Chat", "chat"),
		Users:       users,
		Pagination:  pg,
		ChatEnabled: h.chatEnabled(r),
	})
}

// ChatUserToggleBan blocks or unblocks a chat user from posting. Because chat identity is
// re-read from the DB on every request, the change takes effect on the user's next action.
// The "ban author" button on the messages page posts a return field so it lands back there.
func (h *Handler) ChatUserToggleBan(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	u, err := h.q.GetChatUserByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if u.IsBanned {
		err = h.q.UnbanChatUser(r.Context(), id)
	} else {
		err = h.q.BanChatUser(r.Context(), id)
	}
	if err != nil {
		http.Error(w, "failed to update chat user", http.StatusInternalServerError)
		return
	}
	action, detail := "ban", "Banned chat user "+u.Name
	if u.IsBanned {
		action, detail = "unban", "Unbanned chat user "+u.Name
	}
	h.audit(r, action, "chat_user", &id, detail)
	http.Redirect(w, r, chatReturnDest(r, "/admin/chat/users"), http.StatusSeeOther)
}

// ChatSettingsUpdate saves the global chat kill switch. The checkbox posts "on" when
// enabled; absent means off.
func (h *Handler) ChatSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	enabled := r.FormValue("enabled") == "on"
	v := "1"
	if !enabled {
		v = "0"
	}
	if err := h.q.UpsertSetting(r.Context(), sqlc.UpsertSettingParams{K: chatEnabledKey, V: v}); err != nil {
		http.Error(w, "failed to save chat settings", http.StatusInternalServerError)
		return
	}
	detail := "Enabled public chat"
	if !enabled {
		detail = "Disabled public chat"
	}
	h.audit(r, "update", "chat_settings", nil, detail)
	if enabled {
		h.flash(w, "Public chat is now on.")
	} else {
		h.flash(w, "Public chat is now off.")
	}
	http.Redirect(w, r, "/admin/chat", http.StatusSeeOther)
}

// chatReturnDest returns the posted "return" field if it is a safe in-panel path, else the
// fallback. Restricting to /admin/chat* keeps this from becoming an open redirect.
func chatReturnDest(r *http.Request, fallback string) string {
	dest := r.FormValue("return")
	if dest == "/admin/chat" || dest == "/admin/chat/users" {
		return dest
	}
	return fallback
}
