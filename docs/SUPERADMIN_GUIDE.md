# ClassyFM Superadmin Guide

This guide covers the features available only to accounts with the **Superadmin** role.

Superadmins can do everything a content editor can — those tasks (programs, broadcasters,
news, podcasts, ads, and so on) are documented in the
[Admin Guide](ADMIN_GUIDE.md). In addition, superadmins see extra items others don't:
**Legal Pages** (in the sidebar's **Radio** group), **Contact** (in the **Content**
group), and a **System** group with **Users** and **Activity Log**. Those are described
below.

> **Roles at a glance:** there are two roles — **Admin** (content editors) and
> **Superadmin**. Only superadmins can edit the legal pages and contact details, manage
> user accounts, and view the Activity Log.

## Table of Contents

- [Legal Pages](#1-legal-pages)
- [Contact](#2-contact)
- [Users](#3-users)
- [Activity Log](#4-activity-log)
- [Dashboard "Recent activity" panel](#5-dashboard-recent-activity-panel)

---

## 1. Legal Pages

Edit the **Privacy Policy** and **Terms & Conditions** shown on the public site.
**Sidebar → Radio → Legal Pages.**

The list has two fixed pages — Privacy Policy and Terms & Conditions. Click either to
edit it (there's nothing to create or delete). Each edit form has:

| Field | Notes |
|---|---|
| Title | Required. The large heading in the page's navy banner. |
| Intro | Optional. The subtitle under the heading; leave empty to hide it. |
| Body | The page content, written in a rich **Markdown** editor. |

The **Body** editor has a formatting toolbar; switch to **Markdown** at the bottom of the
editor to edit the raw source instead. Content is stored as Markdown. Click **Save** when
done, or **View public page** to open the live page in a new tab.

These pages also feed the mobile app (via the public API), so keep them current.

---

## 2. Contact

Set the contact details used across the public site — the floating **WhatsApp** button
and the footer's **"Get in touch"** block. **Sidebar → Content → Contact.**

It's a single form; every field is optional, and leaving one blank hides the thing it
powers.

| Field | Notes |
|---|---|
| WhatsApp number | Include the country code (e.g. `+62 812 …`). Blank hides the floating WhatsApp button and the footer's WhatsApp line. Must contain at least a few digits if set. |
| Prefilled message | The message WhatsApp opens with when a visitor taps "Message us". Blank = a blank chat. |
| Phone | Shown as a tappable phone link in the footer. Blank hides it. |
| Email | Shown as a mailto link in the footer. Must look like an address (contain `@`). Blank hides it. |

Click **Save** to apply. These details also feed the mobile app via the public API.

---

## 3. Users

Manage who can sign in to the admin panel. **Sidebar → System → Users.**

The list is searchable and sortable (by name and email) and shows each user's **Role**
(an amber "Superadmin" or gray "Admin" badge) and **Status** (Active / Inactive). Your own
row is marked **Your Account** and has no action buttons — manage your own details from
your [Profile](ADMIN_GUIDE.md#13-your-profile) instead.

### Create a user

1. Click **Add User**.
2. Fill in the fields and click **Save**.

**Fields**

| Field | Notes |
|---|---|
| Name | Required. |
| Email | Required and unique. |
| Password | Required for a new user; minimum 8 characters. |
| Role | **Admin** or **Superadmin**. Superadmins can manage Users and view the Activity Log. |
| Active | Untick to disable the account without deleting it. |

### Edit a user

Open a user with the **Edit** pencil, change what you need, and click **Save**.

- **Password:** leave the "New password" field **blank to keep** the current password.
  Entering a new one resets it **and signs that user out of all their sessions**.
- **Deactivating** a user (unticking Active) also signs them out everywhere and blocks them
  from signing in.

### Delete a user

Click **Delete** and confirm *"Delete this user?"*. You cannot delete your own account from
here.

---

## 4. Activity Log

A read-only, append-only history of changes made in the admin panel — the accountability
trail for who did what. **Sidebar → System → Activity Log.**

The table is searchable (by action, entity, detail or user) and sortable. Each row shows:

| Column | Meaning |
|---|---|
| Time | When the action happened. |
| User | Who did it (their email at the time). |
| Action | What kind of change — create, update, delete, login, logout, ban / unban, hide / un-hide, password reset, password change, feed refresh, and so on. |
| Entity | What was affected, with its type and `#id`. |
| Detail | A short human-readable description. |

The log records IP address per entry. It cannot be edited or deleted — it is a permanent
record.

---

## 5. Dashboard "Recent activity" panel

On the Dashboard, superadmins see an extra **Recent activity** panel (content editors do
not). It lists the most recent Activity Log entries with a "time ago" label and a link to
the full **Activity log** — a quick way to see the latest changes at a glance.
