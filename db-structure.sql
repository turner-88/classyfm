-- Adminer 5.3.0 MariaDB 12.1.2-MariaDB dump

SET NAMES utf8;
SET time_zone = '+00:00';
SET foreign_key_checks = 0;
SET sql_mode = 'NO_AUTO_VALUE_ON_ZERO';

SET NAMES utf8mb4;

DROP TABLE IF EXISTS `about_page_banner`;
CREATE TABLE `about_page_banner` (
  `id` tinyint(3) unsigned NOT NULL DEFAULT 1,
  `media_type` enum('image','video') NOT NULL DEFAULT 'image',
  `image_url` varchar(1000) DEFAULT NULL,
  `video_url` varchar(1000) DEFAULT NULL,
  `updated_at` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;


DROP TABLE IF EXISTS `about_page_segments`;
CREATE TABLE `about_page_segments` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `segment` enum('profile','music','audience') NOT NULL,
  `title` varchar(255) NOT NULL DEFAULT '',
  `body` text NOT NULL,
  `updated_at` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_segment` (`segment`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;


DROP TABLE IF EXISTS `ad_banners`;
CREATE TABLE `ad_banners` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `slot` enum('top','bottom') NOT NULL,
  `title` varchar(255) NOT NULL DEFAULT '',
  `alt_text` varchar(255) NOT NULL DEFAULT '',
  `image_url` varchar(1000) NOT NULL,
  `link_url` varchar(1000) DEFAULT NULL,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL DEFAULT current_timestamp(),
  `updated_at` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `idx_ad_banners_slot_active` (`slot`,`is_active`,`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;


DROP TABLE IF EXISTS `ad_banner_pages`;
CREATE TABLE `ad_banner_pages` (
  `banner_id` bigint(20) unsigned NOT NULL,
  `page` enum('home','about','program','program_detail','live','news','news_detail','broadcasters','broadcaster_detail') NOT NULL,
  PRIMARY KEY (`banner_id`,`page`),
  CONSTRAINT `fk_ad_banner_pages_banner` FOREIGN KEY (`banner_id`) REFERENCES `ad_banners` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;


DROP TABLE IF EXISTS `ad_slots`;
CREATE TABLE `ad_slots` (
  `slot` enum('top','bottom') NOT NULL,
  `label` varchar(100) NOT NULL DEFAULT '',
  `display_mode` enum('stacked','slideshow') NOT NULL DEFAULT 'stacked',
  `rotate_secs` int(11) NOT NULL DEFAULT 6,
  `show_placeholder` tinyint(1) NOT NULL DEFAULT 0,
  `placeholder_text` varchar(255) NOT NULL DEFAULT 'Ad space available',
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `updated_at` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`slot`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;


DROP TABLE IF EXISTS `audit_logs`;
CREATE TABLE `audit_logs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned DEFAULT NULL,
  `user_email_snapshot` varchar(191) NOT NULL DEFAULT '',
  `action` varchar(32) NOT NULL,
  `entity_type` varchar(32) NOT NULL,
  `entity_id` bigint(20) unsigned DEFAULT NULL,
  `detail` text NOT NULL,
  `ip_address` varchar(64) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `idx_audit_created_at` (`created_at`),
  KEY `fk_audit_user` (`user_id`),
  CONSTRAINT `fk_audit_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;


DROP TABLE IF EXISTS `broadcasters`;
CREATE TABLE `broadcasters` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `slug` varchar(255) NOT NULL,
  `role` varchar(255) DEFAULT NULL,
  `photo_url` varchar(1000) DEFAULT NULL,
  `bio` text DEFAULT NULL,
  `birth_place` varchar(255) DEFAULT NULL,
  `birth_date` varchar(20) DEFAULT NULL,
  `instagram` varchar(255) DEFAULT NULL,
  `twitter` varchar(255) DEFAULT NULL,
  `facebook` varchar(255) DEFAULT NULL,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL DEFAULT current_timestamp(),
  `updated_at` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `slug` (`slug`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


DROP TABLE IF EXISTS `feed_sources`;
CREATE TABLE `feed_sources` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `source` enum('youtube','klikpositif','katasumbar') NOT NULL,
  `is_enabled` tinyint(1) NOT NULL DEFAULT 1,
  `endpoint` varchar(1000) DEFAULT NULL,
  `last_fetched_at` datetime DEFAULT NULL,
  `last_status` varchar(255) DEFAULT NULL,
  `item_count` int(11) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `source` (`source`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;


DROP TABLE IF EXISTS `media_links`;
CREATE TABLE `media_links` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `platform` enum('instagram','facebook','x','youtube','spotify','tiktok') NOT NULL,
  `url` varchar(1000) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_platform` (`platform`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;


DROP TABLE IF EXISTS `news_items`;
CREATE TABLE `news_items` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `source` enum('youtube','klikpositif','katasumbar','hot_release') NOT NULL,
  `external_id` varchar(255) DEFAULT NULL,
  `title` varchar(500) NOT NULL,
  `slug` varchar(255) DEFAULT NULL,
  `excerpt` text DEFAULT NULL,
  `content` mediumtext DEFAULT NULL,
  `url` varchar(1000) DEFAULT NULL,
  `image_url` varchar(1000) DEFAULT NULL,
  `thumb_url` varchar(1000) DEFAULT NULL,
  `published_at` datetime NOT NULL,
  `is_published` tinyint(1) NOT NULL DEFAULT 1,
  `is_featured` tinyint(1) NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL DEFAULT current_timestamp(),
  `updated_at` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_source_external` (`source`,`external_id`),
  UNIQUE KEY `uq_slug` (`slug`),
  KEY `idx_feed` (`is_published`,`published_at`),
  KEY `idx_source` (`source`,`published_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;


DROP TABLE IF EXISTS `password_reset_tokens`;
CREATE TABLE `password_reset_tokens` (
  `token_hash` char(64) NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `expires_at` datetime NOT NULL,
  `used_at` datetime DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`token_hash`),
  KEY `fk_reset_user` (`user_id`),
  CONSTRAINT `fk_reset_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;


DROP TABLE IF EXISTS `programs`;
CREATE TABLE `programs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `title` varchar(255) NOT NULL,
  `slug` varchar(255) NOT NULL,
  `description` text DEFAULT NULL,
  `image_url` varchar(1000) DEFAULT NULL,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL DEFAULT current_timestamp(),
  `updated_at` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `slug` (`slug`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;


DROP TABLE IF EXISTS `program_broadcasters`;
CREATE TABLE `program_broadcasters` (
  `program_id` bigint(20) unsigned NOT NULL,
  `broadcaster_id` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`program_id`,`broadcaster_id`),
  KEY `idx_program_broadcasters_broadcaster` (`broadcaster_id`),
  CONSTRAINT `fk_program_broadcasters_broadcaster` FOREIGN KEY (`broadcaster_id`) REFERENCES `broadcasters` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_program_broadcasters_program` FOREIGN KEY (`program_id`) REFERENCES `programs` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;


DROP TABLE IF EXISTS `program_schedules`;
CREATE TABLE `program_schedules` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `program_id` bigint(20) unsigned NOT NULL,
  `day_of_week` tinyint(4) NOT NULL,
  `start_time` time NOT NULL,
  `end_time` time NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_prog` (`program_id`),
  KEY `idx_day` (`day_of_week`,`start_time`),
  CONSTRAINT `fk_sched_prog` FOREIGN KEY (`program_id`) REFERENCES `programs` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;


DROP TABLE IF EXISTS `schedule_broadcasters`;
CREATE TABLE `schedule_broadcasters` (
  `schedule_id` bigint(20) unsigned NOT NULL,
  `broadcaster_id` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`schedule_id`,`broadcaster_id`),
  KEY `idx_schedule_broadcasters_broadcaster` (`broadcaster_id`),
  CONSTRAINT `fk_schedule_broadcasters_broadcaster` FOREIGN KEY (`broadcaster_id`) REFERENCES `broadcasters` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_schedule_broadcasters_schedule` FOREIGN KEY (`schedule_id`) REFERENCES `program_schedules` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;


DROP TABLE IF EXISTS `schema_migrations`;
CREATE TABLE `schema_migrations` (
  `version` bigint(20) NOT NULL,
  `dirty` tinyint(1) NOT NULL,
  PRIMARY KEY (`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


DROP TABLE IF EXISTS `sessions`;
CREATE TABLE `sessions` (
  `token` char(64) NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `expires_at` datetime NOT NULL,
  PRIMARY KEY (`token`),
  KEY `idx_exp` (`expires_at`),
  KEY `fk_sess_user` (`user_id`),
  CONSTRAINT `fk_sess_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;


DROP TABLE IF EXISTS `settings`;
CREATE TABLE `settings` (
  `k` varchar(191) NOT NULL,
  `v` text NOT NULL,
  PRIMARY KEY (`k`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;


DROP TABLE IF EXISTS `users`;
CREATE TABLE `users` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `email` varchar(191) NOT NULL,
  `password_hash` varchar(255) NOT NULL,
  `name` varchar(255) NOT NULL,
  `role` enum('superadmin','admin') NOT NULL DEFAULT 'admin',
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;


-- 2026-07-30 06:12:19 UTC
