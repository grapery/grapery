-- Writers Room Tables Migration
-- Create tables for story writers room chat system

-- writers_rooms: Main chat room table for story collaboration
CREATE TABLE IF NOT EXISTS writers_rooms (
  id VARCHAR(36) PRIMARY KEY,
  story_id VARCHAR(36) NOT NULL,
  title VARCHAR(255) NOT NULL,
  last_message TEXT,
  last_message_time BIGINT,
  message_count INT DEFAULT 0,
  participant_count INT DEFAULT 0,
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL,
  UNIQUE KEY idx_story_id (story_id),
  INDEX idx_updated_at (updated_at),
  FOREIGN KEY (story_id) REFERENCES stories(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- writers_room_participants: Track participants in writers room
CREATE TABLE IF NOT EXISTS writers_room_participants (
  id VARCHAR(36) PRIMARY KEY,
  room_id VARCHAR(36) NOT NULL,
  user_id VARCHAR(36) NOT NULL,
  role ENUM('owner', 'admin', 'member') DEFAULT 'member',
  joined_at BIGINT NOT NULL,
  last_read_at BIGINT DEFAULT 0,
  UNIQUE KEY idx_room_user (room_id, user_id),
  INDEX idx_user_id (user_id),
  INDEX idx_room_id (room_id),
  FOREIGN KEY (room_id) REFERENCES writers_rooms(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- writers_room_messages: Store messages in writers room
CREATE TABLE IF NOT EXISTS writers_room_messages (
  id VARCHAR(36) PRIMARY KEY,
  room_id VARCHAR(36) NOT NULL,
  sender_id VARCHAR(36) NOT NULL,
  content TEXT NOT NULL,
  message_type ENUM('text', 'image', 'mixed', 'system') DEFAULT 'text',
  attachments JSON,
  mentions JSON,
  reply_to_message_id VARCHAR(36),
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL,
  INDEX idx_room_id_created_at (room_id, created_at),
  INDEX idx_sender_id (sender_id),
  INDEX idx_reply_to_message_id (reply_to_message_id),
  FOREIGN KEY (room_id) REFERENCES writers_rooms(id) ON DELETE CASCADE,
  FOREIGN KEY (sender_id) REFERENCES writers_room_participants(id) ON DELETE CASCADE,
  FOREIGN KEY (reply_to_message_id) REFERENCES writers_room_messages(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- writers_room_message_reactions: Store user reactions to messages
CREATE TABLE IF NOT EXISTS writers_room_message_reactions (
  id VARCHAR(36) PRIMARY KEY,
  message_id VARCHAR(36) NOT NULL,
  user_id VARCHAR(36) NOT NULL,
  reaction_type VARCHAR(50) NOT NULL,
  emoji_code VARCHAR(50),
  created_at BIGINT NOT NULL,
  UNIQUE KEY idx_message_user_reaction (message_id, user_id, reaction_type),
  INDEX idx_message_id (message_id),
  INDEX idx_user_id (user_id),
  FOREIGN KEY (message_id) REFERENCES writers_room_messages(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- writers_room_read_receipts: Track when users have read messages
CREATE TABLE IF NOT EXISTS writers_room_read_receipts (
  id VARCHAR(36) PRIMARY KEY,
  message_id VARCHAR(36) NOT NULL,
  user_id VARCHAR(36) NOT NULL,
  read_at BIGINT NOT NULL,
  UNIQUE KEY idx_message_user (message_id, user_id),
  INDEX idx_user_id (user_id),
  INDEX idx_message_id (message_id),
  FOREIGN KEY (message_id) REFERENCES writers_room_messages(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
