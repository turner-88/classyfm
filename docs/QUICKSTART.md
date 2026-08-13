# ClassyFM Admin Quickstart

A short summary of the day-to-day steps to start managing the ClassyFM website through the
admin panel. For full details on every feature, see the **Admin Guide** (`ADMIN_GUIDE`);
for superadmin-only features (Users, Legal Pages, SEO, Activity Log), see the **Superadmin
Guide** (`SUPERADMIN_GUIDE`).

---

## 1. Sign In

1. Open **https://classyfm.co.id/admin/login**.
2. Enter your **Email** and **Password**, then click **Sign In**.
3. Forgot your password? Click **Forgot password?**, enter your registered email, and
   follow the reset link sent to that address.

**Change your own password:** click the account name at the bottom of the sidebar
(**Profile** menu), then the **Change Password** section. **Log Out** is also at the bottom
of the sidebar.

---

## 2. Layout & Common Conventions

- The **left sidebar** holds the menus: **Radio** (Programs, Broadcasters, About Us) and
  **Content** (Hero, Hot Release, Podcast, Event, Newsfeed, Media, Ads). On mobile, the
  sidebar is behind the hamburger button.
- List pages have **Search**, column sorting (click a column header), and **50 rows** per
  page.
- After a successful **Save**/**Delete**, a green confirmation appears. Delete always shows
  a confirmation dialog.
- Image fields have a live preview; images are auto-compressed in the browser
  (JPG/PNG/WEBP/GIF). **On edit forms, leave an image field empty to keep the current
  image.**
- **Active/Published** shows a green badge; **Inactive/Draft/Hidden** shows a grey badge.

---

## 3. Dashboard

The panel's home page: an **On-air** card (current & next programs, stream status, now
playing, listener count), stat tiles, a **Needs attention** list (things to fix), listener
and incoming-news charts, and feed-source health. Quick buttons: **New program**, **New
hot release**, **Refresh feeds**.

---

## 4. Common Daily Tasks

### Programs — **Radio → Programs**
Click **Add Program** (or Edit). Fill in **Title**, description, **Default broadcasters**,
image, and check **Active**. On the edit form, use **Weekly Schedule** → **Add slot** to
set the day, start/end times, and per-slot broadcasters. The system flags overlapping
schedules (a warning, not a blocker).

### Broadcasters — **Radio → Broadcasters**
Click **Add Broadcaster** (or Edit). Fill in **Name**, role, bio, photo (4:5 portrait),
social links, and **Active**. Assignment to programs is managed from the **Programs** menu,
not here.

### Editorial news — **Content → Hot Release**
Click **Add Hot Release**. Fill in **Title**, excerpt, **Content**, cover image, **Publish
Date**, then check **Publish** to make it public. The star icon on the list toggles
featured inline.

### Podcasts — **Content → Podcast**
Click **Add Podcast**. Pick a **Series**, paste a valid **Spotify URL**
(`open.spotify.com`), and check **Publish**. Title, image, player, and description are
pulled automatically from Spotify on save — use **Refresh from Spotify** when an episode's
details change.

### Home page — **Content → Hero**
Manage the homepage banner slides (**Add Slide**): badge, title, excerpt, image
(2400×1200), optional link, **Active**. The **Hero settings** panel controls the mix of
image slides and news.

### Aggregated news — **Content → Newsfeed**
Auto-fetched news from YouTube/KlikPositif/KataSumbar. Toggle **Publish/Hide** (eye icon)
and **Feature** (star icon); **Refresh Feed** pulls the latest. Manage sources via the
**Feed Sources** button.

### Others
- **Content → Event:** public events/promos.
- **Content → Ads:** banner ads for the top/bottom slots.
- **Content → Media:** official social media URLs (leave blank to hide an icon).
- **Radio → About Us:** the About page content.

---

## 5. Next Steps

- Full content management and per-field details → **Admin Guide** (`ADMIN_GUIDE`).
- Adding admin users, legal pages, SEO, and the activity log → **Superadmin Guide**
  (`SUPERADMIN_GUIDE`).
