-- 016_update_notifications_table.sql
-- 更新 notifications 表以支持新的通知类型和字段
-- 基于 StoryCreationAppUI MessagesPage 设计

-- 添加故事相关字段
ALTER TABLE notifications ADD COLUMN story_title VARCHAR(200) DEFAULT '' COMMENT '故事标题';
ALTER TABLE notifications ADD COLUMN story_cover VARCHAR(500) DEFAULT '' COMMENT '故事封面';
ALTER TABLE notifications ADD COLUMN story_id VARCHAR(36) DEFAULT '' COMMENT '故事ID';
ALTER TABLE notifications ADD INDEX idx_story_id (story_id);

-- 添加评论内容字段
ALTER TABLE notifications ADD COLUMN comment_text TEXT COMMENT '评论内容摘要';

-- 添加系统通知字段
ALTER TABLE notifications ADD COLUMN sys_title VARCHAR(200) DEFAULT '' COMMENT '系统通知标题';
ALTER TABLE notifications ADD COLUMN sys_body TEXT COMMENT '系统通知正文';
ALTER TABLE notifications ADD COLUMN sys_icon VARCHAR(50) DEFAULT '' COMMENT '系统通知图标名称';

-- 更新 Type 字段注释以包含新的类型
ALTER TABLE notifications MODIFY COLUMN type VARCHAR(50) NOT NULL COMMENT 'like, comment, follow, story_update, system';
