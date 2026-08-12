-- Contact settings: the WhatsApp number + prefilled message for the floating
-- WhatsApp widget, and the phone/email shown in the footer's "Get in touch" block
-- (all hardcoded in footer.html before this). A typed single-row table, matching
-- hero_settings/about_page_banner rather than the unused k/v `settings` table (see
-- the note in 0027_hero_slides.up.sql). id is TINYINT UNSIGNED, not TINYINT(1), so
-- sqlc.yaml doesn't map the primary key to a Go bool.
CREATE TABLE contact_settings (
  id               TINYINT UNSIGNED PRIMARY KEY DEFAULT 1,
  whatsapp_number  VARCHAR(64)  NOT NULL DEFAULT '',
  whatsapp_message VARCHAR(500) NOT NULL DEFAULT '',
  phone            VARCHAR(128) NOT NULL DEFAULT '',
  email            VARCHAR(191) NOT NULL DEFAULT '',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Seed the single row with the values previously hardcoded in the footer.
INSERT INTO contact_settings (id, whatsapp_number, whatsapp_message, phone, email) VALUES
(1, '+62 812 6601 0340', 'Halo Classy FM, saya ingin bertanya…', '+62 (0751) 74999', 'classyfm@classyfm.co.id');
