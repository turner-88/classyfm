-- Listener history. The Shoutcast server reports its current audience live
-- ({shoutcastBase}/stats?json=1, "currentlisteners") but remembers nothing: its
-- own "peaklisteners" resets on every server restart, so until now the station
-- had no way to answer "how many people listened yesterday". internal/listeners
-- samples the number on an interval and writes both tables below.
--
-- Two tables rather than one, because they answer different questions and age
-- differently:
--
--   listener_stats    one row per station-local day, kept forever (~365 rows a
--                     year). This is what the admin dashboard chart reads.
--   listener_samples  the raw poll trail, pruned to LISTENER_RETENTION. Kept so
--                     an intraday view can be added later without having to
--                     start collecting from scratch.
--
-- The daily row is upserted on every sample rather than derived from the samples
-- at read time, so the peak outlives pruning and the chart stays a single
-- indexed range scan over a tiny table.
CREATE TABLE listener_stats (
  -- Station-local (WIB) date, computed in Go. The MySQL session timezone is not
  -- guaranteed to be Asia/Jakarta, so bucketing with DATE() here would drift the
  -- day boundary - the same reason ListRecentNewsArrivals buckets in Go.
  --
  -- Natural primary key, deliberately not the usual AUTO_INCREMENT id: there is
  -- exactly one row per day and every write is an upsert keyed on it.
  stat_date      DATE PRIMARY KEY,
  peak_listeners INT UNSIGNED NOT NULL DEFAULT 0,
  peak_at        DATETIME NULL,                  -- when the peak was first reached
  last_listeners INT UNSIGNED NOT NULL DEFAULT 0,
  sample_count   INT UNSIGNED NOT NULL DEFAULT 0,
  created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE listener_samples (
  id         BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  sampled_at DATETIME NOT NULL,
  listeners  INT UNSIGNED NOT NULL,
  is_live    TINYINT(1) NOT NULL DEFAULT 1,      -- streamstatus at sample time
  -- Covers the prune's range delete, and any future intraday read.
  KEY idx_listener_samples_sampled_at (sampled_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
