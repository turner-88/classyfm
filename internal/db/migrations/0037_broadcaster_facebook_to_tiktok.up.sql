-- Broadcasters no longer show a Facebook handle; TikTok replaces it. Straight
-- column rename — existing values carry over verbatim (they are stale Facebook
-- handles until an admin edits each broadcaster), matching how the field is a
-- single free-text handle either way.
ALTER TABLE broadcasters CHANGE facebook tiktok VARCHAR(255) NULL;
