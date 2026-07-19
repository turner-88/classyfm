package admin

import (
	"net/http"
	"strconv"

	"github.com/classyfm/classyfm/internal/db/sqlc"
)

const auditPageSize = 50

type auditTrailListData struct {
	Base       baseData
	Logs       []sqlc.AuditLog
	Page       int
	TotalPages int
}

// AuditTrailList renders a paginated, append-only log of admin-panel mutations
// (superadmin-only, see RequireRole in main.go).
func (h *Handler) AuditTrailList(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	total, err := h.q.CountAuditLogs(r.Context())
	if err != nil {
		http.Error(w, "gagal memuat log aktivitas", http.StatusInternalServerError)
		return
	}
	totalPages := int((total + auditPageSize - 1) / auditPageSize)
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	logs, err := h.q.ListAuditLogs(r.Context(), sqlc.ListAuditLogsParams{
		Limit:  auditPageSize,
		Offset: int32((page - 1) * auditPageSize),
	})
	if err != nil {
		http.Error(w, "gagal memuat log aktivitas", http.StatusInternalServerError)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/audit_trail_list", auditTrailListData{
		Base:       h.base(r, "Log Aktivitas", "audit-trail"),
		Logs:       logs,
		Page:       page,
		TotalPages: totalPages,
	})
}
