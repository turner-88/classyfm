package admin

import (
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/classyfm/classyfm/internal/db/sqlc"
	appmw "github.com/classyfm/classyfm/internal/middleware"
)

type usersListData struct {
	Base          baseData
	Users         []sqlc.User
	CurrentUserID uint64
	Pagination    pagination
}

// UsersList renders every admin-panel user (superadmin-only, see RequireRole in main.go).
func (h *Handler) UsersList(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	search, pattern := searchPattern(r)
	sort, dir := parseSort(r, "name", "asc", "name", "email")
	total, err := h.q.CountUsers(r.Context(), sqlc.CountUsersParams{Search: pattern})
	if err != nil {
		http.Error(w, "failed to load users", http.StatusInternalServerError)
		return
	}
	pg := paginate(r, total, "/admin/users", url.Values{"q": {search}, "sort": {sort}, "dir": {dir}})
	users, err := h.q.ListUsers(r.Context(), sqlc.ListUsersParams{
		Search: pattern, Sort: sort, Dir: dir,
		Limit:  adminPageSize,
		Offset: pg.Offset(),
	})
	if err != nil {
		http.Error(w, "failed to load users", http.StatusInternalServerError)
		return
	}
	var currentID uint64
	if u := appmw.CurrentUser(r); u != nil {
		currentID = u.ID
	}
	h.r.Page(w, http.StatusOK, "admin/users_list", usersListData{
		Base:          h.base(r, "Users", "users"),
		Users:         users,
		CurrentUserID: currentID,
		Pagination:    pg,
	})
}

type userFormData struct {
	Base  baseData
	IsNew bool
	User  sqlc.User
	Error string
}

// UserNew renders the create-user form.
func (h *Handler) UserNew(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	h.r.Page(w, http.StatusOK, "admin/users_form", userFormData{
		Base:  h.base(r, "New User", "users"),
		IsNew: true,
		User:  sqlc.User{Role: sqlc.UsersRoleAdmin, IsActive: true},
	})
}

// UserCreate inserts a new admin-panel user.
func (h *Handler) UserCreate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")
	role := sqlc.UsersRole(r.FormValue("role"))
	isActive := r.FormValue("is_active") == "on"

	renderErr := func(status int, msg string) {
		h.r.Page(w, status, "admin/users_form", userFormData{
			Base:  h.base(r, "New User", "users"),
			IsNew: true,
			User:  sqlc.User{Name: name, Email: email, Role: role, IsActive: isActive},
			Error: msg,
		})
	}

	if name == "" || email == "" || password == "" {
		renderErr(http.StatusBadRequest, "Name, email, and password are required.")
		return
	}
	if email == appmw.VirtualRootEmail {
		renderErr(http.StatusBadRequest, "This email is reserved for the system and cannot be used.")
		return
	}
	if role != sqlc.UsersRoleSuperadmin && role != sqlc.UsersRoleAdmin {
		renderErr(http.StatusBadRequest, "Invalid role.")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		renderErr(http.StatusInternalServerError, "Failed to process password.")
		return
	}

	res, err := h.q.CreateUser(r.Context(), sqlc.CreateUserParams{
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
		Role:         role,
		IsActive:     isActive,
	})
	if err != nil {
		renderErr(http.StatusBadRequest, friendlyDBError(err, "Email is already used by another user."))
		return
	}
	id, _ := res.LastInsertId()
	h.audit(r, "create", "user", uint64Ptr(uint64(id)), "Created user "+email)
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// UserEdit renders the edit form for an existing user. Editing yourself here is
// not allowed - self-service goes through the Profile page instead.
func (h *Handler) UserEdit(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if isSelf(r, id) {
		http.Redirect(w, r, "/admin/profile", http.StatusSeeOther)
		return
	}
	user, err := h.q.GetUserByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h.r.Page(w, http.StatusOK, "admin/users_form", userFormData{
		Base:  h.base(r, "Edit User", "users"),
		IsNew: false,
		User:  user,
	})
}

// UserUpdate saves edits to an existing user (name, email, role - password is
// unchanged here, only ever set on create; users change their own password via
// the Profile page).
func (h *Handler) UserUpdate(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if isSelf(r, id) {
		http.Redirect(w, r, "/admin/profile", http.StatusSeeOther)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	role := sqlc.UsersRole(r.FormValue("role"))
	isActive := r.FormValue("is_active") == "on"

	renderErr := func(status int, msg string) {
		h.r.Page(w, status, "admin/users_form", userFormData{
			Base:  h.base(r, "Edit User", "users"),
			IsNew: false,
			User:  sqlc.User{ID: id, Name: name, Email: email, Role: role, IsActive: isActive},
			Error: msg,
		})
	}

	if name == "" || email == "" {
		renderErr(http.StatusBadRequest, "Name and email are required.")
		return
	}
	if email == appmw.VirtualRootEmail {
		renderErr(http.StatusBadRequest, "This email is reserved for the system and cannot be used.")
		return
	}
	if role != sqlc.UsersRoleSuperadmin && role != sqlc.UsersRoleAdmin {
		renderErr(http.StatusBadRequest, "Invalid role.")
		return
	}

	if err := h.q.UpdateUser(r.Context(), sqlc.UpdateUserParams{Name: name, Email: email, Role: role, IsActive: isActive, ID: id}); err != nil {
		renderErr(http.StatusBadRequest, friendlyDBError(err, "Email is already used by another user."))
		return
	}
	if !isActive {
		_ = h.q.DeleteSessionsByUserID(r.Context(), id)
	}
	h.audit(r, "update", "user", &id, "Updated user "+email)
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// UserDelete removes a user. Deleting yourself is not allowed.
func (h *Handler) UserDelete(w http.ResponseWriter, r *http.Request) {
	if h.unavailable(w, r) {
		return
	}
	id, ok := parseIDParam(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if isSelf(r, id) {
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
		return
	}
	user, _ := h.q.GetUserByID(r.Context(), id)
	if err := h.q.DeleteUser(r.Context(), id); err != nil {
		http.Error(w, "failed to delete user", http.StatusInternalServerError)
		return
	}
	h.audit(r, "delete", "user", &id, "Deleted user "+user.Email)
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func isSelf(r *http.Request, id uint64) bool {
	u := appmw.CurrentUser(r)
	return u != nil && u.ID == id
}

func uint64Ptr(v uint64) *uint64 { return &v }
