-- 017_add_characters_views_json.sql
-- 角色三视图 URL 持久化（与 GORM Character.ViewsJSON / domain CharacterThreeViews 对应）

ALTER TABLE characters ADD COLUMN views_json JSON NULL COMMENT '三视图 front/side/back URL JSON';
