package admin

import (
	"net/http"
	"net/url"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

type auditTrailListData struct {
	Base       baseData
	Logs       []sqlc.AuditLog
	Pagination pagination
}

// AuditTrailList renders a paginated, append-only log of admin-panel mutations
// (superadmin-only, see RequireRole in main.go).
func (h *Handler) AuditTrailList(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	search, pattern := searchPattern(r)
	sort, dir := parseSort(r, "created_at", "desc", "created_at", "action", "entity_type")
	total, err := h.q.CountAuditLogs(r.Context(), sqlc.CountAuditLogsParams{Search: pattern})
	if err != nil {
		http.Error(w, "failed to load activity log", http.StatusInternalServerError)
		return
	}
	pg := paginate(r, total, "/admin/audit-trail", url.Values{"q": {search}, "sort": {sort}, "dir": {dir}})
	logs, err := h.q.ListAuditLogs(r.Context(), sqlc.ListAuditLogsParams{
		Search: pattern, Sort: sort, Dir: dir,
		Limit:  adminPageSize,
		Offset: pg.Offset(),
	})
	if err != nil {
		http.Error(w, "failed to load activity log", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/audit_trail_list", auditTrailListData{
		Base:       h.base(r, "Activity Log", "audit-trail"),
		Logs:       logs,
		Pagination: pg,
	})
}
