# Handover Document — ClassyFM Website

Handover record for the development of the **Classy 103.4 FM Padang** website and admin
panel, per the scope agreed in the *Website Development Offer & Proposal*
(`Penawaran_ClassyFM.md`), section 2.8 (Deliverables).

| | |
|---|---|
| **Project** | Classy 103.4 FM Website & Admin Panel |
| **From** | habiburrahman (Web Developer) |
| **To** | Classy 103.4 FM Padang Management |
| **Handover date** | _______________________ |
| **Reference** | Development Offer & Proposal (approved) |

---

## 1. Summary of Deliverables

Per proposal section 2.8, the following three items are delivered:

**1. Complete source code.** All source code for the website and admin panel — a Go
server with server-side HTML rendering, the admin CMS, and all templates and static
assets embedded directly in the binary (self-contained via `go:embed`). Data is stored
in MySQL.

- Repository location: _______________________ *(fill in: Git URL or handover medium)*

**2. Live production website.** The application is deployed and running on the production
server.

- Public site: **https://classyfm.co.id**
- Admin panel: **https://classyfm.co.id/admin**

**3. Handover document & usage guides.** This document plus a set of supporting guides
(available as Markdown and PDF under `docs/`):

| Document | Contents |
|---|---|
| **Quickstart** (`QUICKSTART`) | Day-to-day operational steps to get started. |
| **Admin Guide** (`ADMIN_GUIDE`) | Full content-management guide for content editors. |
| **Superadmin Guide** (`SUPERADMIN_GUIDE`) | Superadmin-only features (Users, Legal, SEO, Activity Log). |
| **API Reference** (`API`) | Public JSON API documentation for the mobile app. |

---

## 2. Admin Account Credentials

An initial admin account (role **Superadmin**) has been created for Classy 103.4 FM.

| Field | Value |
|---|---|
| Panel URL | https://classyfm.co.id/admin/login |
| Email | _______________________ |
| Password | _______________________ |
| Role | Superadmin |

> **Important — change the password immediately.** After the first login, change the
> password via the **Profile** menu (click the account name at the bottom of the sidebar,
> then the **Change Password** section; or open `/admin/profile`). Minimum 8 characters.
> Store the credentials securely and do not share them.

Additional admin accounts for other team members can be created via the **Users** menu
(superadmin only) — see the Superadmin Guide.

---

## 3. Operator Note (Break-glass Access)

The system includes a built-in emergency (break-glass) login using the reserved email
`root@local.system`, whose password is the server's `SESSION_SECRET` value. This login has
no row in the users table and works even when the database is unavailable. It is intended
for the server operator/administrator in emergencies (e.g. recovering access), **not for
day-to-day use**, and is not the credential handed over in section 2 above. Rotating
`SESSION_SECRET` invalidates any active break-glass login and impersonation sessions.

---

## 4. Infrastructure & Deployment Summary

- **Application:** a single self-contained binary — templates and static assets are
  embedded, keeping deployment simple with minimal dependencies.
- **Database:** MySQL.
- **Server service:** runs as a `systemd` service on the production host.
- **Backups:** automated scheduled data backups (`systemd` timer).
- **Network:** the site sits behind Cloudflare; CSS/JS changes may require a cache purge
  to appear immediately.
- **Image uploads:** program/broadcaster/news images are stored on the server disk and
  served under `/uploads/`.

---

## 5. Scope & Limitations

**Included in this handover:**

- Development of the public site and admin panel per the proposal scope.
- Deployment to the production server.
- Functional testing before handover.
- This handover document, the basic usage guide, and admin account credentials.

**Not included (Classy 103.4 FM's responsibility, unless otherwise agreed):**

- Domain, hosting/server, and SSL certificate costs.
- Maintenance and post-handover support — can be proposed separately if needed.
- Development of new features beyond the agreed scope.

---

## 6. Handover Acceptance

By signing this document, both parties confirm that the work has been delivered and
accepted per the agreed scope.

**Delivered by (Developer):**

Name: habiburrahman

Title: Web Developer

Signature / Date: _______________________

<br>

**Accepted by (Classy 103.4 FM):**

Name: _______________________

Title: _______________________

Signature / Date: _______________________
