# ClassyFM Admin Guide

A practical guide to the ClassyFM admin panel for content editors. It covers everyday
tasks: signing in, managing programs and broadcasters, publishing news and podcasts,
running the home page, and keeping the site's links up to date.

> **Note for superadmins:** if your account has the **Superadmin** role you will see extra
> items others don't — **Legal Pages** (Radio group), **Contact** (Content group), and a
> **System** group with **Users** and **Activity Log**. Those features are documented
> separately in the [Superadmin Guide](SUPERADMIN_GUIDE.md). Everything in *this* guide
> applies to you too.

## Table of Contents

- [Getting started](#1-getting-started)
- [Things that work the same everywhere](#2-things-that-work-the-same-everywhere)
- [Dashboard](#3-dashboard)
- [Programs](#4-programs)
- [Broadcasters](#5-broadcasters)
- [Hero (home page slider)](#6-hero-home-page-slider)
- [Hot Release (news articles)](#7-hot-release-news-articles)
- [Podcasts](#8-podcasts)
- [Newsfeed (aggregated news)](#9-newsfeed-aggregated-news)
- [Ads](#10-ads)
- [Media links](#11-media-links)
- [About Us](#12-about-us)
- [Your profile](#13-your-profile)

---

## 1. Getting started

### Signing in

1. Go to `/admin/login`.
2. Enter your **Email** and **Password** and click **Sign In**.

If the email or password is wrong you'll see "Incorrect email or password." Deactivated
accounts cannot sign in — contact a superadmin if you think your account is disabled.

### Forgot your password?

1. On the login page click **Forgot password?**
2. Enter your email and click **Send Link**.
3. If the address is registered, a reset link is emailed to you. Open it, choose a new
   password (at least 8 characters) and confirm it.
4. After resetting, you're returned to the login page with a confirmation and can sign in
   with the new password.

For security the panel always says "If that address is registered, a reset link has been
sent" — whether or not the account exists. Resetting a password signs you out of every
device.

### The screen layout

- **Sidebar (left):** the main navigation. The logo at the top returns you to the
  Dashboard. Menu items are grouped:
  - **Dashboard** (top)
  - **Radio:** Programs, Broadcasters, About Us
  - **Content:** Hero, Hot Release, Podcast, Newsfeed, Media, Ads
- **Your account (bottom of sidebar):** your name (click it to open your **Profile**) and
  a **Log Out** button.
- On mobile the sidebar is hidden behind a hamburger (menu) button.

### Logging out

Click **Log Out** at the bottom of the sidebar. You are logged out automatically after 7
days.

---

## 2. Things that work the same everywhere

Once you know these, every section behaves predictably.

- **Search:** most list pages have a search box. Type a term and click **Search**; click
  **Reset** to clear it. No matches shows `No results for "…"`.
- **Sorting:** table headers that are links can be clicked to sort. An arrow shows the
  active column — ▲ ascending, ▼ descending. Click again to reverse.
- **Pagination:** lists show 50 rows per page. When there's more than one page you'll see
  **Page X of Y** with **Previous** / **Next** buttons.
- **Saving:** after a successful save or delete you're returned to the list with a green
  confirmation message at the top (for example "Program saved.").
- **Errors:** if something is invalid, the form reloads with a red message explaining what
  to fix. Your entries are kept.
- **Deleting:** delete buttons ask you to confirm in a browser popup before anything is
  removed.
- **Unsaved changes:** if you try to leave an edit form with unsaved changes, your browser
  warns you before navigating away.
- **Images:** image fields show a live preview. Pictures are automatically resized and
  compressed in your browser before uploading (you may briefly see "Compressing…").
  Accepted types are JPG, PNG, WEBP and GIF, and each field shows the recommended
  dimensions. **On an edit form, leave the image field empty to keep the current image** —
  you only need to choose a file when you want to replace it.

Throughout the panel, **Active / Published** items are shown with a green badge and
**Inactive / Draft / Hidden** items with a gray badge.

---

## 3. Dashboard

The Dashboard is the landing page and a live overview of the station. It refreshes parts
of itself automatically.

- **Quick actions (top):** **New program**, **New hot release**, and **Refresh feeds**
  (fetches the latest aggregated news right now).
- **On-air card:** shows the current weekday, date and station clock, whether the stream
  is up, what's on air now (with a progress bar), and what's up next. The **Now playing**
  panel shows the current song, how many people are listening now and today's peak, and an
  **Open the live page** link.
- **Stat tiles:** clickable cards for Programs, Broadcasters, Hot Release and the newsfeed
  — each links straight to that section.
- **Needs attention:** a triage list that flags things to fix — failing or disabled feed
  sources, unpublished news awaiting review, active programs with no airtime or no
  broadcaster, overlapping schedule slots (two shows booked at the same time), ad slots
  enabled with no banner, and blank social links. Each item links to the page where you
  can fix it. When there's nothing to do it reads "All clear."
- **Charts:** a listeners-over-time chart (switch between Daily / Hourly / 5-minute
  views), a "News arriving" chart of items per day by source, and a **This week on air**
  schedule map (blocks link to the program's schedule; a red line marks "now").
- **Feed sources:** a health summary per source (Healthy / Failing / Disabled) with the
  last fetch time and item count.

---

## 4. Programs

Programs are the shows on the schedule. **Sidebar → Radio → Programs.**

### Create or edit a program

1. Click **Add Program** (or the **Edit** pencil on a row).
2. Fill in the fields (see below) and click **Save**.

On an existing program a **View public page** button opens the live page.

**Fields**

| Field | Notes |
|---|---|
| Title | Required. |
| Slug | Required, unique. Used in the public URL `/program/slug`. |
| Description | Free text. |
| Default broadcasters | Multi-select (searchable checkbox list). Used for any schedule slot that doesn't set its own hosts. |
| Program Image | Recommended 1600×900 (16:9). |
| Order | Lower numbers appear first. |
| Active | Untick to hide the program from the public site. |

### Weekly schedule

On the edit form there's a **Weekly Schedule** editor:

1. Click **Add slot** for each broadcast slot.
2. Set the **Day**, **Start** and **End** time (HH:MM).
3. Optionally set **Broadcasters** for that slot. Leave it as **Program default** to use
   the program's default broadcasters instead.
4. Remove a slot with its trash button. Click **Save** to store the whole schedule.

Each slot needs a valid day and a start time different from its end time.

### Schedule conflicts

The panel watches for slots that overlap in time and points them out so you don't
accidentally book two shows at once:

- **On the Programs list**, a warning banner appears at the top whenever any slots overlap
  anywhere across the station, spelling out each clash — for example *"Monday: 'Show A'
  (08:00–10:00) overlaps 'Show B' (09:00–11:00)."* If there are more than ten, the rest are
  summarised as a "+N more" line.
- **On a program's edit form**, the same kind of warning shows just the clashes that
  involve the program you're editing — including a program whose own two slots overlap each
  other.

These are **warnings, not blocks** — you can still save. They're there to help you catch a
mistake, so review them and adjust the times if a clash wasn't intended.

### Delete a program

Click the **Delete** (trash) button and confirm *"Delete this program along with its
schedule?"* — this also removes the program's schedule slots.

---

## 5. Broadcasters

The on-air announcers and hosts. **Sidebar → Radio → Broadcasters.** Create, edit and
delete work exactly like Programs (**Add Broadcaster**, Edit pencil, Delete → *"Delete
this broadcaster?"*).

**Fields**

| Field | Notes |
|---|---|
| Name | Required. |
| Slug | Required, unique. Used in the public URL. |
| Role | e.g. "Announcer". |
| Bio | Free text. |
| Birthplace / Date of Birth | Optional. |
| Instagram / X / Facebook | Optional social links. |
| Photo | Recommended 1000×1250 (4:5 portrait). |
| Order | Lower numbers appear first. |
| Active | Untick to hide from the public site. |

The edit form shows an **Appears On** list of programs this broadcaster is assigned to.
This is read-only here — you change program assignments from the **Program** side (the
program's *Default broadcasters* field or a schedule slot's broadcasters).

---

## 6. Hero (home page slider)

The rotating hero at the top of the home page mixes your image slides with the latest
news. **Sidebar → Content → Hero.**

### Manage slides

Create/edit/delete slides just like other sections (**Add Slide**, Edit, Delete →
*"Delete this hero slide?"*).

**Slide fields**

| Field | Notes |
|---|---|
| Badge | Small label shown on the slide. |
| Title | Slide heading. |
| Excerpt | Short supporting text. |
| Image | Required. Recommended 2400×1200 (full-bleed). |
| Link | Optional. A full `http(s)` URL or a `/relative` path. |
| Open the link in a new tab | Optional. |
| Order | Lower numbers appear first. |
| Active | Untick to hide the slide. |

### Hero settings

Below the list, open the **Hero settings** panel to control how slides and news are mixed:

- **Slide order:** Newest first / Random / News first / Image slides first.
- **News items** (0–10) and **Image slides** (0–10): how many of each to include. Set
  either count to 0 to show only the other type. Click **Save hero settings**.

---

## 7. Hot Release (news articles)

Hot Release is the station's own editorial news. **Sidebar → Content → Hot Release.**

### Create or edit an article

1. Click **Add Hot Release** (or Edit).
2. Fill in the fields and click **Save**.

**Fields**

| Field | Notes |
|---|---|
| Title | Required. |
| Slug | Required, unique. Used in the URL `/news/…`. |
| Excerpt | Short summary. |
| Content | Plain text — any HTML tags are stripped automatically on save. |
| Hot Release Image | Cover image. Recommended 1600×900. |
| Middle Images | An in-article gallery (see below). |
| Publish Date | Required (date + time). |
| Publish | Tick to make it public. |
| Featured | Tick to highlight it. |

**Middle Images gallery:** add several images to appear inside the article. Reorder
existing images by dragging the handle or using the **Move up / Move down** arrows, tick
**Remove** to drop one, and use the multi-file picker to add new images to the end.

**Feature toggle from the list:** each row has a star — click it to feature/unfeature the
article without opening it. Delete confirms *"Delete this Hot Release?"*.

---

## 8. Podcasts

Podcast episodes pulled from Spotify. **Sidebar → Content → Podcast.**

### Add or edit a podcast

1. Click **Add Podcast** (or Edit).
2. Fill in the fields and click **Save**.

**Fields**

| Field | Notes |
|---|---|
| Title | Required. The slug is generated automatically from the title. |
| Series | Required. Choose from the dropdown (see Podcast Series below). |
| Spotify URL | Required. A valid `open.spotify.com` link — the title, thumbnail, player and description are pulled from it when you save. |
| Description | Read-only. Fetched automatically from the Spotify link on save (and refreshed whenever you change the Spotify URL or use **Refresh from Spotify**) — there's no field to type into. |
| Broadcasters | Optional multi-select. |
| Publish | Tick to make it public. |

The **Title**, **Thumbnail** and **Description** are all fetched from Spotify automatically
on save, so you don't upload or type them.

**Refresh from Spotify:** on the edit form there's a **Refresh from Spotify** button. Click
it to re-pull the title, thumbnail and description from the Spotify link right away — handy
when the episode's details changed on Spotify but its URL didn't (they also refresh on
their own whenever you change the Spotify URL).

### Podcast Series

Series group podcasts and power the filter chips on the public podcast page. Reach them via
the **Series** button on the Podcast list.

- **Add Series** / Edit / Delete.
- **Fields:** Name (required; slug generated automatically), Sort order, and **Active**
  (tick = visible on the public site).
- Inactive series keep a gray **Inactive** badge in the list and are hidden from the
  public podcast filter chips — their existing podcasts still appear in the full list.
- You can't delete a series that still has podcasts assigned to it — reassign those
  podcasts to another series first.

---

## 9. Newsfeed (aggregated news)

The Newsfeed shows articles pulled automatically from **YouTube**, **KlikPositif** and
**KataSumbar**. You don't create or edit this content — you only decide what appears
publicly. **Sidebar → Content → Newsfeed.**

- **Filter** with the pills at the top: All / YouTube / KlikPositif / KataSumbar.
- **Publish / Hide** (eye icon): toggle whether an item shows on the public site.
- **Feature** (star icon): toggle whether an item is highlighted.
- **Refresh Feed:** fetch the latest items now.
- Titles link out to the original article.

### Feed Sources

Reach this from the **Feed Sources** button on the Newsfeed page. It shows each source's
health (last status, last fetch time, item count). In one form you can:

- Tick/untick **Active** to enable or disable a source.
- Set an **Endpoint** override (channel ID or RSS URL); leave it empty to use the built-in
  default.
- Click **Save** to store changes, or **Refresh Now** to fetch immediately.

One broken source never blocks the others.

---

## 10. Ads

Banner ads shown in two fixed places: a **top** slot and a **bottom** slot.
**Sidebar → Content → Ads.**

### Slot settings

Each slot has an **Ads settings** panel:

- **Display:** Stacked (all banners at once) or Slideshow (rotate).
- **Rotate every (sec):** 2–60, used in slideshow mode.
- **Placeholder text** and **Show placeholder when empty:** what to show when the slot has
  no active banner.
- **Slot enabled:** turn the whole slot on or off.

Click **Save ads settings** to store slot changes.

### Banners

1. Click **Add Banner** (or the "Add banner here" link inside a slot).
2. Fill in the fields and click **Save**.

**Fields**

| Field | Notes |
|---|---|
| Title | Internal label. |
| Alt text | Description for accessibility. |
| Link | Optional URL — opens in a new tab. |
| Image | Required. Recommended 1200×150 (leaderboard); it's letterboxed, not cropped. |
| Targeting | Tick **All pages**, or untick it and choose specific pages. |
| Placement | Which slot (top / bottom). |
| Order | Lower numbers appear first. |
| Active | Untick to hide the banner. |

Delete confirms *"Delete this banner?"*.

---

## 11. Media links

The station's official account URL for each platform, shown as icons in the site header
and footer. **Sidebar → Content → Media.**

There's one field each for **Instagram, Facebook, X, YouTube, Spotify** and **TikTok**.
Enter a full `http(s)` URL, or leave a field **blank to hide that icon**. Click **Save**.
Each row shows a status badge (Connected / Not set) and an open-link button to test the
saved URL.

---

## 12. About Us

The content of the public About Us page. **Sidebar → Radio → About Us.** It's a single
form:

- **Banner:** choose **Image** (upload) or **Video URL** (a YouTube link). Switching
  between the two keeps whatever you had stored for the other.
- **Text segments:** three sections (profile, music, audience), each with a **Title** and
  **Body**. Separate paragraphs in the body with a blank line.

Click **Save**.

---

## 13. Your profile

Manage your own account. **Click your name at the bottom of the sidebar** (or go to
`/admin/profile`). There are two separate forms:

- **Account Details:** update your **Name** and **Email** (your role is shown but not
  editable here). Click **Save**.
- **Change Password:** enter a **New Password** (at least 8 characters) and confirm it,
  then click **Change Password**.
