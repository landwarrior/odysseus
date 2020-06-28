CREATE TABLE `mst_hr` (
  `hr_cd` varchar(16) COLLATE utf8mb4_bin NOT NULL,
  `user_name` varchar(128) COLLATE utf8mb4_bin NOT NULL,
  `name_kana` varchar(128) COLLATE utf8mb4_bin DEFAULT NULL,
  `pw` varchar(128) COLLATE utf8mb4_bin DEFAULT NULL,
  `is_admin` int(11) NOT NULL DEFAULT '0',
  `remarks` varchar(256) COLLATE utf8mb4_bin DEFAULT NULL,
  `bp_id` int(11) DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`hr_cd`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `mst_bp` (
  `bp_id` int(11) NOT NULL,
  `name` varchar(128) COLLATE utf8mb4_bin NOT NULL,
  `name_kana` varchar(256) COLLATE utf8mb4_bin DEFAULT NULL,
  `address` varchar(512) COLLATE utf8mb4_bin DEFAULT NULL,
  `phone` varchar(16) COLLATE utf8mb4_bin DEFAULT NULL,
  `email` varchar(128) COLLATE utf8mb4_bin DEFAULT NULL,
  `remarks` varchar(256) COLLATE utf8mb4_bin DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`bp_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `mst_department` (
  `department_id` int(11) NOT NULL,
  `name` varchar(64) COLLATE utf8mb4_bin NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`department_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `mst_department_hr` (
  `department_id` int(11) NOT NULL,
  `hr_cd` varchar(16) COLLATE utf8mb4_bin NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`department_id`,`hr_cd`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `mst_role` (
  `role_id` int(11) NOT NULL,
  `name` varchar(32) COLLATE utf8mb4_bin NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `mst_hr_unit_price` (
  `hr_cd` varchar(16) COLLATE utf8mb4_bin NOT NULL,
  `from_date` date NOT NULL,
  `role_id` int(11) NOT NULL,
  `price` int(11) NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`hr_cd`,`from_date`,`role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `mst_process` (
  `process_id` int(11) NOT NULL,
  `name` varchar(32) COLLATE utf8mb4_bin NOT NULL,
  `role_id` int(11) DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`process_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `trn_project` (
  `project_no` varchar(32) COLLATE utf8mb4_bin NOT NULL,
  `name` varchar(64) COLLATE utf8mb4_bin NOT NULL,
  `order_amount` int(11) DEFAULT NULL,
  `from_date` date DEFAULT NULL,
  `to_date` date DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`project_no`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `trn_project_detail` (
  `project_no` varchar(32) COLLATE utf8mb4_bin NOT NULL,
  `process_id` int(11) NOT NULL,
  `from_date` date DEFAULT NULL,
  `to_date` date DEFAULT NULL,
  `man_per_day` int(11) DEFAULT NULL,
  `pre_cost` int(11) DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`project_no`,`process_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `trn_project_detail_hr` (
  `project_no` varchar(32) COLLATE utf8mb4_bin NOT NULL,
  `process_id` int(11) NOT NULL,
  `hr_cd` varchar(16) COLLATE utf8mb4_bin NOT NULL,
  `role_id` int(11) NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`project_no`,`process_id`,`hr_cd`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE `trn_hr_result` (
  `project_no` varchar(32) COLLATE utf8mb4_bin NOT NULL,
  `process_id` int(11) NOT NULL,
  `hr_cd` varchar(16) COLLATE utf8mb4_bin NOT NULL,
  `target_date` date NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`project_no`,`process_id`,`hr_cd`,`target_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
