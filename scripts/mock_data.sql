-- ============================================================
-- Grapery Mock Data for UI Testing
-- Database: grapery
-- ============================================================
-- Usage (use Makefile DB config):
--   make mock-load
-- Or manually:
--   mysql -u $(DB_USERNAME) -p$(DB_PASSWORD) -h $(DB_ADDRESS) $(DB_DATABASE) < scripts/mock_data.sql
-- Default: mysql -u root -p12345678 -h localhost grapery < scripts/mock_data.sql
-- ============================================================

-- Clear existing data (be careful in production!)
SET FOREIGN_KEY_CHECKS = 0;

-- Disable safe updates
SET SQL_SAFE_UPDATES = 0;

-- Clear tables
DELETE FROM notifications WHERE 1=1;
DELETE FROM comments WHERE 1=1;
DELETE FROM fragment_comments WHERE 1=1;
DELETE FROM fragment_likes WHERE 1=1;
DELETE FROM fragment_shares WHERE 1=1;
DELETE FROM fragment_generation_tasks WHERE 1=1;
DELETE FROM fragments WHERE 1=1;
DELETE FROM storyboard_video_generations WHERE 1=1;
DELETE FROM storyboard_image_generations WHERE 1=1;
DELETE FROM storyboard_scene_generations WHERE 1=1;
DELETE FROM storyboard_content_generations WHERE 1=1;
DELETE FROM storyboard_scenes WHERE 1=1;
DELETE FROM storyboard_scene_links WHERE 1=1;
DELETE FROM storyboard_character_links WHERE 1=1;
DELETE FROM storyboards WHERE 1=1;
DELETE FROM story_scenes WHERE 1=1;
DELETE FROM characters WHERE 1=1;
DELETE FROM panels WHERE 1=1;
DELETE FROM story_contributors WHERE 1=1;
DELETE FROM stories WHERE 1=1;
DELETE FROM likes WHERE 1=1;
DELETE FROM follows WHERE 1=1;
DELETE FROM user_follows WHERE 1=1;
DELETE FROM story_likes WHERE 1=1;
DELETE FROM story_follows WHERE 1=1;
DELETE FROM character_follows WHERE 1=1;
DELETE FROM user_settings WHERE 1=1;
DELETE FROM memberships WHERE 1=1;
DELETE FROM invitation_codes WHERE 1=1;
DELETE FROM user_referrals WHERE 1=1;
DELETE FROM user_login_records WHERE 1=1;
DELETE FROM third_party_logins WHERE 1=1;
DELETE FROM tags WHERE 1=1;
DELETE FROM story_tags WHERE 1=1;
DELETE FROM character_tags WHERE 1=1;
DELETE FROM style_configs WHERE 1=1;
DELETE FROM agents WHERE 1=1;
DELETE FROM ai_generation_records WHERE 1=1;
DELETE FROM view_histories WHERE 1=1;
DELETE FROM search_histories WHERE 1=1;
DELETE FROM assets WHERE 1=1;
DELETE FROM users WHERE 1=1;

SET SQL_SAFE_UPDATES = 1;
SET FOREIGN_KEY_CHECKS = 1;

-- ============================================================
-- 1. Users (用户表)
-- ============================================================

INSERT INTO users (id, username, email, password_hash, display_name, avatar, background, bio, location, website, followers, following, storyboard_count, fragments_count, status, email_verified, last_login_at, points, referral_code, created_at, updated_at) VALUES
-- 测试用户 1: 内容创作者 Alice
('user-0001-0000-0000-000000000001', 'creator_alice', 'alice@test.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZRGdjGj/n3.QGhS5ThOy6CxbLWPPm', 'Alice 创作者', 'https://picsum.photos/seed/alice/200/200', 'https://picsum.photos/seed/alice-bg/800/400', '热爱创作奇幻故事，专注于角色塑造和世界观构建', '上海', 'https://alice-portfolio.com', 1250, 89, 15, 42, 'active', 1, UNIX_TIMESTAMP() * 1000 - 3600000, 5000, 'ALICE2024', FROM_UNIXTIME(UNIX_TIMESTAMP() - 2592000), FROM_UNIXTIME(UNIX_TIMESTAMP() - 3600)),

-- 测试用户 2: 普通用户 Bob
('user-0001-0000-0000-000000000002', 'reader_bob', 'bob@test.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZRGdjGj/n3.QGhS5ThOy6CxbLWPPm', 'Bob 读者', 'https://picsum.photos/seed/bob/200/200', 'https://picsum.photos/seed/bob-bg/800/400', '喜欢阅读各种类型的故事，尤其喜欢科幻和悬疑', '北京', '', 320, 156, 2, 0, 'active', 1, UNIX_TIMESTAMP() * 1000 - 7200000, 1200, 'BOB2024', FROM_UNIXTIME(UNIX_TIMESTAMP() - 1728000), FROM_UNIXTIME(UNIX_TIMESTAMP() - 7200)),

-- 测试用户 3: 协作者 Charlie
('user-0001-0000-0000-000000000003', 'writer_charlie', 'charlie@test.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZRGdjGj/n3.QGhS5ThOy6CxbLWPPm', 'Charlie 编剧', 'https://picsum.photos/seed/charlie/200/200', 'https://picsum.photos/seed/charlie-bg/800/400', '专业编剧，擅长剧情设计和对话写作', '深圳', 'https://charlie-writes.com', 890, 234, 8, 12, 'active', 1, UNIX_TIMESTAMP() * 1000 - 1800000, 3500, 'CHARLIE24', FROM_UNIXTIME(UNIX_TIMESTAMP() - 1296000), FROM_UNIXTIME(UNIX_TIMESTAMP() - 1800)),

-- 测试用户 4: 新用户 Diana
('user-0001-0000-0000-000000000004', 'newbie_diana', 'diana@test.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZRGdjGj/n3.QGhS5ThOy6CxbLWPPm', 'Diana 新手', 'https://picsum.photos/seed/diana/200/200', 'https://picsum.photos/seed/diana-bg/800/400', '刚加入社区，正在学习如何创作故事', '杭州', '', 15, 8, 0, 3, 'active', 0, UNIX_TIMESTAMP() * 1000 - 86400000, 100, 'DIANA2024', FROM_UNIXTIME(UNIX_TIMESTAMP() - 172800), FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400)),

-- 测试用户 5: VIP用户 Eve
('user-0001-0000-0000-000000000005', 'vip_eve', 'eve@test.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZRGdjGj/n3.QGhS5ThOy6CxbLWPPm', 'Eve VIP', 'https://picsum.photos/seed/eve/200/200', 'https://picsum.photos/seed/eve-bg/800/400', '资深创作者，VIP会员，喜欢尝试各种创作风格', '广州', 'https://eve-creates.com', 5600, 420, 45, 128, 'active', 1, UNIX_TIMESTAMP() * 1000 - 600000, 15000, 'EVEVIP24', FROM_UNIXTIME(UNIX_TIMESTAMP() - 5184000), FROM_UNIXTIME(UNIX_TIMESTAMP() - 600));

-- ============================================================
-- 2. User Settings (用户设置)
-- ============================================================

INSERT INTO user_settings (id, user_id, language, theme, font_size, data_saver, profile_visibility, default_story_visibility, default_fragment_visibility, allow_follow_from, allow_comments_from, allow_messages_from, show_online_status, show_read_receipts, ai_enabled, a_idata_sharing, notification_settings, updated_at) VALUES
(UUID(), 'user-0001-0000-0000-000000000001', 'zh-CN', 'dark', 'medium', 0, 'public', 'public', 'public', 'everyone', 'everyone', 'followers_only', 1, 1, 1, 1, '{"push":true,"email":true,"likes":true,"comments":true,"follows":true}', UNIX_TIMESTAMP()),
(UUID(), 'user-0001-0000-0000-000000000002', 'zh-CN', 'system', 'medium', 0, 'public', 'public', 'public', 'everyone', 'everyone', 'everyone', 1, 1, 1, 0, '{"push":true,"email":false,"likes":true,"comments":true,"follows":true}', UNIX_TIMESTAMP()),
(UUID(), 'user-0001-0000-0000-000000000003', 'zh-CN', 'light', 'large', 0, 'public', 'public', 'followers_only', 'followers', 'followers', 'followers_only', 0, 1, 1, 1, '{"push":true,"email":true,"likes":true,"comments":true,"follows":true}', UNIX_TIMESTAMP()),
(UUID(), 'user-0001-0000-0000-000000000004', 'zh-CN', 'system', 'medium', 1, 'public', 'public', 'public', 'everyone', 'everyone', 'everyone', 1, 1, 1, 1, '{"push":true,"email":true,"likes":true,"comments":true,"follows":true}', UNIX_TIMESTAMP()),
(UUID(), 'user-0001-0000-0000-000000000005', 'en-US', 'dark', 'medium', 0, 'public', 'public', 'public', 'everyone', 'everyone', 'everyone', 1, 1, 1, 1, '{"push":true,"email":true,"likes":true,"comments":true,"follows":true}', UNIX_TIMESTAMP());

-- ============================================================
-- 3. Stories (故事表)
-- ============================================================

INSERT INTO stories (id, title, description, cover_image, author_id, likes, followers, saves, panels, storyboard_count, default_scene_count, genre, style, status, is_collaboration_open, visibility, use_ai, ai_enabled, created_at, updated_at) VALUES
-- 故事 1: 奇幻冒险
('story-0001-0000-0000-000000000001', '龙之谷传说', '在一个被遗忘的古老大陆上，年轻的勇士踏上了寻找传说中龙之谷的冒险之旅。他将面对重重考验，结识各路伙伴，最终揭开这片土地尘封已久的秘密。', 'https://picsum.photos/seed/dragon-valley/400/600', 'user-0001-0000-0000-000000000001', 856, 234, 156, 0, 5, 4, 'fantasy', '{"artStyle":"anime","colorPalette":"vibrant","lighting":"dramatic"}', 'published', 1, 'public', 1, 1, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2160000), FROM_UNIXTIME(UNIX_TIMESTAMP() - 432000)),

-- 故事 2: 科幻史诗
('story-0001-0000-0000-000000000002', '星际迷航：新纪元', '公元3000年，人类已经掌握了星际旅行技术。一艘探索飞船在新发现的星系中遭遇神秘信号，船员们必须做出抉择：是返回安全地带，还是深入未知？', 'https://picsum.photos/seed/space-odyssey/400/600', 'user-0001-0000-0000-000000000001', 1234, 456, 289, 0, 8, 5, 'scifi', '{"artStyle":"realistic","colorPalette":"cool","lighting":"cinematic"}', 'published', 0, 'public', 1, 1, FROM_UNIXTIME(UNIX_TIMESTAMP() - 1728000), FROM_UNIXTIME(UNIX_TIMESTAMP() - 259200)),

-- 故事 3: 悬疑推理
('story-0001-0000-0000-000000000003', '午夜列车', '一列深夜行驶的列车上，乘客们一个接一个消失。私家侦探林默必须在天亮之前找出真凶，否则下一个消失的可能就是他自己。', 'https://picsum.photos/seed/midnight-train/400/600', 'user-0001-0000-0000-000000000003', 567, 123, 45, 0, 1, 3, 'mystery', '{"artStyle":"noir","colorPalette":"dark","lighting":"shadowy"}', 'published', 0, 'public', 1, 1, FROM_UNIXTIME(UNIX_TIMESTAMP() - 1209600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 259200)),

-- 故事 4: 咖啡馆邂逅
('story-0001-0000-0000-000000000005', '咖啡馆邂逅', '在一个温暖的午后，两个陌生人在咖啡馆相遇。他们的对话从一本书开始，逐渐展开成一段意想不到的故事。', 'https://picsum.photos/seed/cafe-meet/400/600', 'user-0001-0000-0000-000000000005', 2345, 567, 345, 0, 3, 3, 'romance', '{"artStyle":"watercolor","colorPalette":"warm","lighting":"soft"}', 'published', 0, 'public', 1, 1, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2505600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400));

-- ============================================================
-- 4. Characters (角色表)
-- ============================================================

INSERT INTO characters (id, story_id, name, description, avatar, author_id, personality, background, short_term_goal, long_term_goal, handling_style, cognition_range, ability_features, appearance, dress_preference, source_type, created_by, is_public, created_at, updated_at) VALUES
-- 龙之谷传说角色
('char-0001-0000-0000-000000000001', 'story-0001-0000-0000-000000000001', '艾瑞克', '年轻的冒险者，拥有与生俱来的勇气和正义感。在一次偶然的机会中得到了一张古老地图，从此踏上了寻找龙之谷的旅程。', 'https://picsum.photos/seed/eric/200/200', 'user-0001-0000-0000-000000000001', '["勇敢","善良","正义","好奇心强"]', '出生于边境小镇的普通家庭，从小听着龙之谷的传说长大。父亲是退伍军人，母亲是药剂师。', '找到龙之谷的入口', '揭开龙之谷的秘密，成为传说中的英雄', '遇到困难时会先观察分析，再采取行动。不轻易放弃。', '对世界充满好奇，喜欢学习新事物。对历史和传说特别感兴趣。', '剑术基础、野外生存技能、地图解读能力', '金色短发，蓝色眼睛。身材匀称，常穿简单的冒险服装。', '实用主义的皮甲和斗篷，随身携带地图和匕首', 'manual', 'user-0001-0000-0000-000000000001', 1, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2160000), FROM_UNIXTIME(UNIX_TIMESTAMP() - 432000)),

('char-0001-0000-0000-000000000002', 'story-0001-0000-0000-000000000001', '莉娜', '神秘的精灵少女，守护着通往龙之谷必经之路的精灵森林。对外来者充满戒备，但对艾瑞克产生了好奇。', 'https://picsum.photos/seed/lina/200/200', 'user-0001-0000-0000-000000000001', '["神秘","谨慎","善良","独立"]', '精灵族的后裔，从小接受守护森林的使命。对外界充满好奇但被禁止离开森林。', '判断艾瑞克是否值得信任', '探索森林之外的世界，寻找自己的命运', '谨慎而敏锐，善于观察他人的真实意图。行动迅速果断。', '对森林中的一切了如指掌，对人类世界知之甚少但充满好奇。', '精灵魔法、隐身术、与动物沟通', '银色长发，翠绿色眼睛。身材纤细，举止优雅。', '精灵风格的轻便服饰，带有自然元素装饰', 'manual', 'user-0001-0000-0000-000000000001', 1, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 432000)),

-- 星际迷航角色
('char-0001-0000-0000-000000000003', 'story-0001-0000-0000-000000000002', '陈星河', '探索者号的舰长，冷静睿智的领导者。面对未知时总能做出正确的判断，深受船员信任。', 'https://picsum.photos/seed/chen/200/200', 'user-0001-0000-0000-000000000001', '["冷静","睿智","负责任","果断"]', '太空舰队学院优秀毕业生，多次执行深空任务。对宇宙的未知充满敬畏和好奇。', '解读神秘信号并做出正确决策', '带领船员安全返回，同时推进人类对宇宙的认知', '善于倾听不同意见，但最终决策果断。在危机中保持冷静。', '对宇宙科学有深入了解，善于从细节中发现关键信息。', '星舰指挥、战略规划、危机处理', '黑色短发。深邃的眼睛。身材挺拔，常穿深蓝色舰长制服。', '整洁的舰长制服，胸前佩戴勋章', 'manual', 'user-0001-0000-0000-000000000001', 1, FROM_UNIXTIME(UNIX_TIMESTAMP() - 1728000), FROM_UNIXTIME(UNIX_TIMESTAMP() - 259200)),

-- 午夜列车角色
('char-0001-0000-0000-000000000004', 'story-0001-0000-0000-000000000003', '林默', '私家侦探，观察力敏锐，擅长推理分析。曾是一名优秀的刑警，因一次案件失败而辞职。', 'https://picsum.photos/seed/linmo/200/200', 'user-0001-0000-0000-000000000003', '["敏锐","冷静","独立","正义"]', '前刑警，因一次案件失败而辞职成为私家侦探。', '找出列车上的凶手', '赎清过去的失败，重建信任', '习惯从细节入手，善于发现被忽视的线索。', '对犯罪心理学有深入研究，善于从细节中发现关键信息。', '推理分析、格斗术、枪械使用', '黑色中长发，深邃的眼睛。常穿黑色风衣。', '复古风格的黑色风衣，灰色围巾', 'manual', 'user-0001-0000-0000-000000000003', 1, FROM_UNIXTIME(UNIX_TIMESTAMP() - 1209600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 259200)),

-- 咖啡馆邂逅角色
('char-0001-0000-0000-000000000005', 'story-0001-0000-0000-000000000005', '苏晓雨', '年轻设计师，热爱生活，追求自由和梦想。刚从设计学院毕业，正在为自己的工作室努力。', 'https://picsum.photos/seed/xiaoyu/200/200', 'user-0001-0000-0000-000000000005', '["活泼","善良","独立","有梦想"]', '刚从设计学院毕业，正在为自己的工作室努力。', '完成第一个独立设计项目', '成为知名的独立设计师', '面对挑战时会积极寻找机会，对时尚和艺术趋势有敏锐嗅觉。', '对时尚和艺术趋势有敏锐嗅觉，善于发现美。', '服装设计、手绘、色彩搭配', '栗色长发，明亮的大眼睛，甜美的笑容。', '简约时尚的个人风格，喜欢复古元素', 'manual', 'user-0001-0000-0000-000000000005', 1, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2505600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400));

-- ============================================================
-- 5. Story Scenes (故事场景)
-- ============================================================

INSERT INTO story_scenes (id, story_id, title, description, image, location, time_of_day, source_type, created_by, is_public, tags, created_at, updated_at) VALUES
-- 龙之谷传说场景
('scene-0001-0000-0000-000000000001', 'story-0001-0000-0000-000000000001', '边境小镇', '艾瑞克长大的地方，宁静祥和的小镇，远处是连绵的山脉。', 'https://picsum.photos/seed/border-town/800/450', '边境小镇', 'day', 'manual', 'user-0001-0000-0000-000000000001', 1, '["城镇","起点","家乡"]', FROM_UNIXTIME(UNIX_TIMESTAMP() - 2160000), FROM_UNIXTIME(UNIX_TIMESTAMP() - 432000)),
('scene-0001-0000-0000-000000000002', 'story-0001-0000-0000-000000000001', '精灵森林', '神秘的古老森林，阳光透过树叶洒下斑驳的光影。', 'https://picsum.photos/seed/elf-forest/800/450', '精灵森林', 'day', 'manual', 'user-0001-0000-0000-000000000001', 1, '["森林","神秘","自然"]', FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 432000)),
('scene-0001-0000-0000-000000000003', 'story-0001-0000-0000-000000000001', '龙之谷入口', '传说中的入口，巨大的石门上刻满了古老的符文。', 'https://picsum.photos/seed/dragon-gate/800/450', '龙之谷', 'sunset', 'manual', 'user-0001-0000-0000-000000000001', 1, '["遗迹","神秘","关键地点"]', FROM_UNIXTIME(UNIX_TIMESTAMP() - 1900800), FROM_UNIXTIME(UNIX_TIMESTAMP() - 432000)),

-- 星际迷航场景
('scene-0001-0000-0000-000000000004', 'story-0001-0000-0000-000000000002', '探索者号舰桥', '飞船的指挥中心，巨大的全息屏幕显示着星空。', 'https://picsum.photos/seed/starship-bridge/800/450', '探索者号', 'night', 'manual', 'user-0001-0000-0000-000000000001', 1, '["科幻","太空","飞船"]', FROM_UNIXTIME(UNIX_TIMESTAMP() - 1728000), FROM_UNIXTIME(UNIX_TIMESTAMP() - 259200)),
('scene-0001-0000-0000-000000000005', 'story-0001-0000-0000-000000000002', '未知星球', '新发现的神秘星球，紫色天空下是奇异的植物。', 'https://picsum.photos/seed/alien-planet/800/450', '未知星球', 'day', 'manual', 'user-0001-0000-0000-000000000001', 1, '["外星","神秘","探索"]', FROM_UNIXTIME(UNIX_TIMESTAMP() - 1555200), FROM_UNIXTIME(UNIX_TIMESTAMP() - 259200));

-- ============================================================
-- 6. Storyboards (故事板)
-- ============================================================

INSERT INTO storyboards (id, story_id, parent_id, creator_id, title, content, raw_input, is_standalone, is_ai_generated, scene_count, workflow_status, current_step, likes, comments, shares, fork_count, views, token_consumption, created_at, updated_at) VALUES
-- 龙之谷传说故事板 (根节点 parent_id 为 NULL)
('board-0001-0000-0000-000000000001', 'story-0001-0000-0000-000000000001', null, 'user-0001-0000-0000-000000000001', '第一章：启程', '艾瑞克站在边境小镇的山丘上，眺望着远方被云雾笼罩的山脉。村里的长者曾告诉他，那些山的深处就是传说中龙之谷的所在。今天，他终于决定踏上这段未知的旅程。

"我一定会找到它的，"艾瑞克握紧了手中的古地图，"龙之谷的秘密，等着我来揭开。"

他深吸一口气，迈出了离开家乡的第一步。阳光洒在他的背上，影子被拉得很长很长。身后是熟悉的家园，前方是未知的命运。', '艾瑞克离开家乡，踏上寻找龙之谷的冒险之旅', 0, 1, 4, 'published', 5, 234, 45, 12, 3, 1567, 2500, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 432000)),

-- 第二章 (parent_id 指向第一章)
('board-0001-0000-0000-000000000002', 'story-0001-0000-0000-000000000001', 'board-0001-0000-0000-000000000001', 'user-0001-0000-0000-000000000001', '第二章：精灵森林', '穿越茂密的森林，艾瑞克感到有目光在注视着自己。突然，一道银色的身影从树间闪过。

"你是谁？为什么来到这里？"清冷的声音从身后传来。

艾瑞克转身，看到了一位美丽的精灵少女正用警惕的目光打量着他。她的银色长发在阳光下闪闪发光，翠绿的眼眸中既有好奇也有戒备。

"我叫艾瑞克，我要去龙之谷。"

"龙之谷？"精灵少女的眼睛亮了起来，"那里...我也一直想去看看。"', '艾瑞克在精灵森林遇到了莉娜', 0, 1, 4, 'published', 5, 189, 34, 8, 2, 1234, 2100, FROM_UNIXTIME(UNIX_TIMESTAMP() - 1987200), FROM_UNIXTIME(UNIX_TIMESTAMP() - 345600)),
-- 平行宇宙分支 (parent_id 指向第一章)
('board-0001-0000-0000-000000000003', 'story-0001-0000-0000-000000000001', 'board-0001-0000-0000-000000000001', 'user-0001-0000-0000-000000000003', '第二章（平行）：商队同行', '艾瑞克决定先加入一支商队，积累经验和资源。在商队中，他结识了各路旅者，听说了更多关于龙之谷的传说——有人说那里藏着无尽的宝藏，也有人说那里是被诅咒的地方。

"年轻人，龙之谷不是闹着玩的，"一位老商人警告道，"去过那里的人，没有一个回来的。"

艾瑞克握紧了拳头，"那我更要去了。有些秘密，值得用生命去揭开。"', '艾瑞克选择加入商队，走另一条路', 1, 1, 3, 'published', 5, 56, 12, 3, 1, 345, 800, FROM_UNIXTIME(UNIX_TIMESTAMP() - 1728000), FROM_UNIXTIME(UNIX_TIMESTAMP() - 518400)),

-- 星际迷航故事板 (根节点 parent_id 为 NULL)
('board-0001-0000-0000-000000000004', 'story-0001-0000-0000-000000000002', null, 'user-0001-0000-0000-000000000001', '序幕：神秘信号', '探索者号在常规巡航中接收到了一组异常信号。舰长陈星河站在舰桥上，看着全息屏幕上闪烁的数据流。

"舰长，这个信号...它来自一个从未被记录的星系，"科学官报告道，"而且...它似乎是故意发送给我们的。"

陈星河皱起眉头，"你是说，有人知道我们会经过这里？"

"不，舰长，"科学官的声音有些颤抖，"我是说...这个信号，是用地球的语言发送的。"

全舰一片死寂。', '探索者号接收到神秘信号', 0, 1, 5, 'published', 5, 345, 67, 23, 5, 2345, 3200, FROM_UNIXTIME(UNIX_TIMESTAMP() - 1641600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 259200));

-- ============================================================
-- 7. Storyboard Scenes (故事板场景)
-- ============================================================

INSERT INTO storyboard_scenes (id, storyboard_id, story_scene_id, sequence, title, description, image, video_url, location, time_of_day, characters, mood, is_ai_generated, created_at, updated_at) VALUES
-- 第一章场景
('sbs-0001-0000-0000-000000000001', 'board-0001-0000-0000-000000000001', 'scene-0001-0000-0000-000000000001', 1, '启程之日', '清晨的阳光洒在边境小镇，艾瑞克背起行囊，最后看了一眼熟悉的街道。村民们纷纷出来送行，祝福声此起彼伏。', 'https://picsum.photos/seed/scene1-1/800/450', '', '边境小镇', 'morning', '["艾瑞克"]', 'hopeful', 1, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600)),
('sbs-0001-0000-0000-000000000002', 'board-0001-0000-0000-000000000001', NULL, 2, '告别', '艾瑞克与父母拥抱告别。父亲递给他一把旧剑，那是他年轻时用过的。母亲塞给他一包药草和干粮。', 'https://picsum.photos/seed/scene1-2/800/450', '', '边境小镇', 'morning', '["艾瑞克"]', 'bittersweet', 1, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600)),
('sbs-0001-0000-0000-000000000003', 'board-0001-0000-0000-000000000001', NULL, 3, '踏上旅途', '艾瑞克走出小镇，眼前是广阔的原野和远方的山脉。他深吸一口气，迈出了冒险的第一步。', 'https://picsum.photos/seed/scene1-3/800/450', '', '原野', 'day', '["艾瑞克"]', 'adventurous', 1, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600));

-- ============================================================
-- 8. Fragments (碎片)
-- ============================================================

INSERT INTO fragments (id, creator_id, content, image_urls, visibility, source_type, is_converted, likes, comments, shares, views, created_at, updated_at) VALUES
-- Alice 的碎片
('frag-0001-0000-0000-000000000001', 'user-0001-0000-0000-000000000001', '今天突然想到一个超棒的故事点子：如果时间可以倒流，但每次只能倒流一分钟，会发生什么有趣的故事呢？想象一下，你可以在说出那句后悔的话之前，倒流回去收回它。或者在考试时，倒流一分钟回忆刚才看过的答案。但是...如果倒流的时候遇到了同样在倒流的另一个人呢？', '["https://picsum.photos/seed/frag1/400/300"]', 'public', 'original', 0, 89, 23, 5, 1234, UNIX_TIMESTAMP() - 864000, UNIX_TIMESTAMP() - 864000),

('frag-0001-0000-0000-000000000002', 'user-0001-0000-0000-000000000001', '【角色设定】一个失去记忆但拥有超能力的少年，他必须在自己的过去追上自己之前，找回丢失的记忆。每当使用超能力，他就会失去一部分现有的记忆。这种代价让他的每次选择都充满了挣扎。', '["https://picsum.photos/seed/frag2/400/300"]', 'public', 'original', 0, 156, 45, 12, 2345, UNIX_TIMESTAMP() - 691200, UNIX_TIMESTAMP() - 691200),

-- Bob 的碎片（收藏的内容）
('frag-0001-0000-0000-000000000003', 'user-0001-0000-0000-000000000002', '分享一个我超喜欢的故事片段：从龙之谷传说的第二章，艾瑞克和莉娜初次相遇的场景。写得真的太美了！精灵少女从树间闪过，那种神秘感扑面而来。', '["https://picsum.photos/seed/frag3/400/300"]', 'public', 'story_excerpt', 0, 45, 12, 8, 567, UNIX_TIMESTAMP() - 432000, UNIX_TIMESTAMP() - 432000),

-- Charlie 的碎片
('frag-0001-0000-0000-000000000004', 'user-0001-0000-0000-000000000003', '【对话练习】\n\n"你知道为什么星星会眨眼睛吗？"\n\n"因为它们也在寻找答案。"\n\n"什么答案？"\n\n"关于存在的意义。"\n\n这是我今天写的一段对话，觉得意境很美，分享给大家。', '[]', 'public', 'original', 0, 234, 67, 34, 3456, UNIX_TIMESTAMP() - 604800, UNIX_TIMESTAMP() - 604800),

-- Diana 的碎片（新用户）
('frag-0001-0000-0000-000000000005', 'user-0001-0000-0000-000000000004', '大家好！我是新来的，刚开始学习创作，请多多指教~ 这是我的第一个角色设定草稿，还不太成熟，希望大家能给些建议！\n\n角色名：小月\n性格：温柔但内心坚强\n背景：失去双亲的孤儿，被一位神秘老人收养', '["https://picsum.photos/seed/frag5/400/300"]', 'public', 'original', 0, 23, 8, 2, 123, UNIX_TIMESTAMP() - 172800, UNIX_TIMESTAMP() - 172800);

-- ============================================================
-- 9. Likes (点赞)
-- ============================================================

INSERT INTO likes (id, user_id, likeable_type, likeable_id, created_at) VALUES
-- Bob 对故事点赞
(UUID(), 'user-0001-0000-0000-000000000002', 'story', 'story-0001-0000-0000-000000000001', UNIX_TIMESTAMP() - 777600),
(UUID(), 'user-0001-0000-0000-000000000002', 'story', 'story-0001-0000-0000-000000000002', UNIX_TIMESTAMP() - 604800),
-- Charlie 对故事点赞
(UUID(), 'user-0001-0000-0000-000000000003', 'story', 'story-0001-0000-0000-000000000001', UNIX_TIMESTAMP() - 1296000),
-- Diana 对故事点赞
(UUID(), 'user-0001-0000-0000-000000000004', 'story', 'story-0001-0000-0000-000000000001', UNIX_TIMESTAMP() - 172800),
-- Eve 对故事点赞
(UUID(), 'user-0001-0000-0000-000000000005', 'story', 'story-0001-0000-0000-000000000001', UNIX_TIMESTAMP() - 1728000),
(UUID(), 'user-0001-0000-0000-000000000005', 'story', 'story-0001-0000-0000-000000000002', UNIX_TIMESTAMP() - 1555200),
-- 对故事板点赞
(UUID(), 'user-0001-0000-0000-000000000002', 'storyboard', 'board-0001-0000-0000-000000000001', UNIX_TIMESTAMP() - 777600),
(UUID(), 'user-0001-0000-0000-000000000003', 'storyboard', 'board-0001-0000-0000-000000000001', UNIX_TIMESTAMP() - 1296000),
-- 对碎片点赞
(UUID(), 'user-0001-0000-0000-000000000002', 'fragment', 'frag-0001-0000-0000-000000000001', UNIX_TIMESTAMP() - 864000),
(UUID(), 'user-0001-0000-0000-000000000003', 'fragment', 'frag-0001-0000-0000-000000000002', UNIX_TIMESTAMP() - 691200);

-- ============================================================
-- 10. Follows (关注)
-- ============================================================

INSERT INTO follows (id, follower_id, followable_type, followable_id, notifications_enabled, created_at) VALUES
-- Bob 关注了 Alice
(UUID(), 'user-0001-0000-0000-000000000002', 'user', 'user-0001-0000-0000-000000000001', 1, UNIX_TIMESTAMP() - 864000),
-- Bob 关注了 Charlie
(UUID(), 'user-0001-0000-0000-000000000002', 'user', 'user-0001-0000-0000-000000000003', 1, UNIX_TIMESTAMP() - 691200),
-- Charlie 关注了 Alice
(UUID(), 'user-0001-0000-0000-000000000003', 'user', 'user-0001-0000-0000-000000000001', 1, UNIX_TIMESTAMP() - 1296000),
-- Diana 关注了 Alice
(UUID(), 'user-0001-0000-0000-000000000004', 'user', 'user-0001-0000-0000-000000000001', 1, UNIX_TIMESTAMP() - 172800),
-- Diana 关注了 Eve
(UUID(), 'user-0001-0000-0000-000000000004', 'user', 'user-0001-0000-0000-000000000005', 1, UNIX_TIMESTAMP() - 86400),
-- Eve 关注了 Alice
(UUID(), 'user-0001-0000-0000-000000000005', 'user', 'user-0001-0000-0000-000000000001', 1, UNIX_TIMESTAMP() - 1728000),
-- Eve 关注了 Charlie
(UUID(), 'user-0001-0000-0000-000000000005', 'user', 'user-0001-0000-0000-000000000003', 1, UNIX_TIMESTAMP() - 1555200),
-- 对故事的关注
(UUID(), 'user-0001-0000-0000-000000000002', 'story', 'story-0001-0000-0000-000000000001', 1, UNIX_TIMESTAMP() - 777600),
(UUID(), 'user-0001-0000-0000-000000000003', 'story', 'story-0001-0000-0000-000000000001', 1, UNIX_TIMESTAMP() - 1296000),
-- 对角色的关注
(UUID(), 'user-0001-0000-0000-000000000002', 'character', 'char-0001-0000-0000-000000000001', 1, UNIX_TIMESTAMP() - 777600);

-- ============================================================
-- 11. Comments (评论)
-- ============================================================

INSERT INTO comments (id, author_id, content, target_type, target_id, likes, reply_count, created_at, updated_at) VALUES
-- 对龙之谷传说的评论
('comm-0001-0000-0000-000000000001', 'user-0001-0000-0000-000000000002', '这个故事太精彩了！艾瑞克的冒险让人热血沸腾，期待后续更新！', 'story', 'story-0001-0000-0000-000000000001', 23, 2, FROM_UNIXTIME(UNIX_TIMESTAMP() - 777600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 777600)),
('comm-0001-0000-0000-000000000002', 'user-0001-0000-0000-000000000003', '角色塑造非常到位，尤其是莉娜这个角色，神秘又吸引人。', 'story', 'story-0001-0000-0000-000000000001', 15, 1, FROM_UNIXTIME(UNIX_TIMESTAMP() - 1296000), FROM_UNIXTIME(UNIX_TIMESTAMP() - 1296000)),
-- 对故事板的评论
('comm-0001-0000-0000-000000000003', 'user-0001-0000-0000-000000000002', '第二章的平行宇宙分支创意太棒了！商队这条线也很期待', 'storyboard', 'board-0001-0000-0000-000000000003', 8, 0, FROM_UNIXTIME(UNIX_TIMESTAMP() - 1728000), FROM_UNIXTIME(UNIX_TIMESTAMP() - 1728000)),
-- 对碎片的评论
('comm-0001-0000-0000-000000000004', 'user-0001-0000-0000-000000000002', '这个时间倒流的设定太有意思了！期待看到完整故事', 'fragment', 'frag-0001-0000-0000-000000000001', 12, 1, FROM_UNIXTIME(UNIX_TIMESTAMP() - 864000), FROM_UNIXTIME(UNIX_TIMESTAMP() - 864000)),
('comm-0001-0000-0000-000000000005', 'user-0001-0000-0000-000000000005', '欢迎 Diana！创作是一个循序渐进的过程，加油！', 'fragment', 'frag-0001-0000-0000-000000000005', 5, 0, FROM_UNIXTIME(UNIX_TIMESTAMP() - 172800), FROM_UNIXTIME(UNIX_TIMESTAMP() - 172800));

-- ============================================================
-- 12. Notifications (通知)
-- ============================================================

INSERT INTO notifications (id, user_id, type, title, content, link, `read`, actor_id, actor_name, actor_avatar, story_title, story_cover, story_id, comment_text, sys_title, sys_body, sys_icon, created_at) VALUES
-- Alice 收到的通知
(UUID(), 'user-0001-0000-0000-000000000001', 'follow', '新粉丝', 'Bob 读者 关注了你', '/user/user-0001-0000-0000-000000000002', 0, 'user-0001-0000-0000-000000000002', 'Bob 读者', 'https://picsum.photos/seed/bob/200/200', NULL, NULL, NULL, NULL, NULL, NULL, NULL, FROM_UNIXTIME(UNIX_TIMESTAMP() - 864000)),
(UUID(), 'user-0001-0000-0000-000000000001', 'like', '收到点赞', 'Bob 读者 喜欢了你的故事《龙之谷传说》', '/story/story-0001-0000-0000-000000000001', 0, 'user-0001-0000-0000-000000000002', 'Bob 读者', 'https://picsum.photos/seed/bob/200/200', '龙之谷传说', 'https://picsum.photos/seed/dragon-valley/400/600', 'story-0001-0000-0000-000000000001', NULL, NULL, NULL, NULL, FROM_UNIXTIME(UNIX_TIMESTAMP() - 777600)),
(UUID(), 'user-0001-0000-0000-000000000001', 'comment', '新评论', 'Bob 读者 评论了你的故事《龙之谷传说》：「这个故事太精彩了！艾瑞克的冒险让人热血沸腾」', '/story/story-0001-0000-0000-000000000001', 1, 'user-0001-0000-0000-000000000002', 'Bob 读者', 'https://picsum.photos/seed/bob/200/200', '龙之谷传说', 'https://picsum.photos/seed/dragon-valley/400/600', 'story-0001-0000-0000-000000000001', '这个故事太精彩了！艾瑞克的冒险让人热血沸腾', NULL, NULL, NULL, FROM_UNIXTIME(UNIX_TIMESTAMP() - 1728000)),
(UUID(), 'user-0001-0000-0000-000000000001', 'system', '创作大赛开始啦！', '参加本月创作大赛，赢取丰厚奖励！', NULL, 0, NULL, NULL, NULL, NULL, NULL, NULL, NULL, '创作大赛开始啦！', '参加本月创作大赛，赢取丰厚奖励！', 'gift', FROM_UNIXTIME(UNIX_TIMESTAMP() - 432000)),
(UUID(), 'user-0001-0000-0000-000000000001', 'like', '收到点赞', 'Charlie 编剧 喜欢了你的碎片「时间倒流一分钟」', '/fragment/frag-0001-0000-0000-000000000001', 0, 'user-0001-0000-0000-000000000003', 'Charlie 编剧', 'https://picsum.photos/seed/charlie/200/200', NULL, NULL, NULL, NULL, NULL, NULL, NULL, FROM_UNIXTIME(UNIX_TIMESTAMP() - 691200)),
(UUID(), 'user-0001-0000-0000-000000000001', 'comment', '新评论', 'Bob 读者 评论了你的碎片：「这个时间倒流的设定太有意思了！期待看到完整故事」', '/fragment/frag-0001-0000-0000-000000000001', 0, 'user-0001-0000-0000-000000000002', 'Bob 读者', 'https://picsum.photos/seed/bob/200/200', NULL, NULL, NULL, '这个时间倒流的设定太有意思了！期待看到完整故事', NULL, NULL, NULL, FROM_UNIXTIME(UNIX_TIMESTAMP() - 860000)),
-- Bob 收到的通知
(UUID(), 'user-0001-0000-0000-000000000002', 'follow', '新粉丝', 'Diana 新手 关注了你', '/user/user-0001-0000-0000-000000000004', 1, 'user-0001-0000-0000-000000000004', 'Diana 新手', 'https://picsum.photos/seed/diana/200/200', NULL, NULL, NULL, NULL, NULL, NULL, NULL, FROM_UNIXTIME(UNIX_TIMESTAMP() - 172800)),
(UUID(), 'user-0001-0000-0000-000000000002', 'update', '故事更新', 'Alice 创作者 更新了《龙之谷传说》第二章', '/story/story-0001-0000-0000-000000000001', 0, 'user-0001-0000-0000-000000000001', 'Alice 创作者', 'https://picsum.photos/seed/alice/200/200', '龙之谷传说', 'https://picsum.photos/seed/dragon-valley/400/600', 'story-0001-0000-0000-000000000001', NULL, NULL, NULL, NULL, FROM_UNIXTIME(UNIX_TIMESTAMP() - 3600)),
(UUID(), 'user-0001-0000-0000-000000000002', 'system', '推荐内容', '你关注的 Alice 创作者 发布了新碎片「记忆晶石」', NULL, 0, NULL, NULL, NULL, NULL, NULL, NULL, NULL, '推荐内容', '你关注的创作者发布了新内容', 'trending', FROM_UNIXTIME(UNIX_TIMESTAMP() - 518400)),
-- Charlie 收到的通知
(UUID(), 'user-0001-0000-0000-000000000003', 'like', '收到点赞', 'Eve VIP 喜欢了你的故事板「第二章（平行）：商队同行」', '/storyboard/board-0001-0000-0000-000000000003', 0, 'user-0001-0000-0000-000000000005', 'Eve VIP', 'https://picsum.photos/seed/eve/200/200', '龙之谷传说', 'https://picsum.photos/seed/dragon-valley/400/600', 'story-0001-0000-0000-000000000001', NULL, NULL, NULL, NULL, FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400)),
(UUID(), 'user-0001-0000-0000-000000000003', 'comment', '新评论', 'Bob 读者 评论了你的故事板：「第二章的平行宇宙分支创意太棒了！商队这条线也很期待」', '/storyboard/board-0001-0000-0000-000000000003', 0, 'user-0001-0000-0000-000000000002', 'Bob 读者', 'https://picsum.photos/seed/bob/200/200', '龙之谷传说', 'https://picsum.photos/seed/dragon-valley/400/600', 'story-0001-0000-0000-000000000001', '第二章的平行宇宙分支创意太棒了！商队这条线也很期待', NULL, NULL, NULL, FROM_UNIXTIME(UNIX_TIMESTAMP() - 1728000)),
-- Diana 收到的通知（新用户引导）
(UUID(), 'user-0001-0000-0000-000000000004', 'system', '欢迎加入 Grapery！', '欢迎来到 Grapery！开始你的创作之旅吧！关注喜欢的创作者，发现精彩故事。', NULL, 0, NULL, NULL, NULL, NULL, NULL, NULL, NULL, '欢迎加入', '欢迎来到 Grapery！开始你的创作之旅吧！关注喜欢的创作者，发现精彩故事。', 'star', FROM_UNIXTIME(UNIX_TIMESTAMP() - 172800)),
(UUID(), 'user-0001-0000-0000-000000000004', 'comment', '新评论', 'Eve VIP 评论了你的碎片：「欢迎 Diana！创作是一个循序渐进的过程，加油！」', '/fragment/frag-0001-0000-0000-000000000005', 0, 'user-0001-0000-0000-000000000005', 'Eve VIP', 'https://picsum.photos/seed/eve/200/200', NULL, NULL, NULL, '欢迎 Diana！创作是一个循序渐进的过程，加油！', NULL, NULL, NULL, FROM_UNIXTIME(UNIX_TIMESTAMP() - 172800)),
-- Eve 收到的通知
(UUID(), 'user-0001-0000-0000-000000000005', 'follow', '新粉丝', 'Diana 新手 关注了你', '/user/user-0001-0000-0000-000000000004', 0, 'user-0001-0000-0000-000000000004', 'Diana 新手', 'https://picsum.photos/seed/diana/200/200', NULL, NULL, NULL, NULL, NULL, NULL, NULL, FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400)),
(UUID(), 'user-0001-0000-0000-000000000005', 'like', '收到点赞', 'Bob 读者 喜欢了你的故事《咖啡馆邂逅》', '/story/story-0001-0000-0000-000000000005', 0, 'user-0001-0000-0000-000000000002', 'Bob 读者', 'https://picsum.photos/seed/bob/200/200', '咖啡馆邂逅', 'https://picsum.photos/seed/cafe-meet/400/600', 'story-0001-0000-0000-000000000005', NULL, NULL, NULL, NULL, FROM_UNIXTIME(UNIX_TIMESTAMP() - 43200)),
(UUID(), 'user-0001-0000-0000-000000000005', 'ai_complete', 'AI 生成完成', '你的碎片「苏晓雨与陌生人对话」已生成完成', '/fragment/frag-0001-0000-0000-000000000007', 0, NULL, NULL, NULL, NULL, NULL, NULL, NULL, 'AI 生成完成', '你的碎片已生成完成，快去看看吧', 'sparkles', FROM_UNIXTIME(UNIX_TIMESTAMP() - 345600));

-- ============================================================
-- 13. Tags (标签) - 使用固定 ID 便于 story_tags/character_tags 引用
-- ============================================================

INSERT INTO tags (id, name, category, usage_count, created_at) VALUES
('tag-0001-0000-0000-000000000001', '奇幻', 'genre', 45, FROM_UNIXTIME(UNIX_TIMESTAMP() - 8640000)),
('tag-0001-0000-0000-000000000002', '科幻', 'genre', 38, FROM_UNIXTIME(UNIX_TIMESTAMP() - 8640000)),
('tag-0001-0000-0000-000000000003', '悬疑', 'genre', 29, FROM_UNIXTIME(UNIX_TIMESTAMP() - 8640000)),
('tag-0001-0000-0000-000000000004', '爱情', 'genre', 56, FROM_UNIXTIME(UNIX_TIMESTAMP() - 8640000)),
('tag-0001-0000-0000-000000000005', '冒险', 'theme', 34, FROM_UNIXTIME(UNIX_TIMESTAMP() - 7776000)),
('tag-0001-0000-0000-000000000006', '时间旅行', 'theme', 23, FROM_UNIXTIME(UNIX_TIMESTAMP() - 6912000)),
('tag-0001-0000-0000-000000000007', '太空', 'theme', 18, FROM_UNIXTIME(UNIX_TIMESTAMP() - 6048000)),
('tag-0001-0000-0000-000000000008', '治愈', 'mood', 41, FROM_UNIXTIME(UNIX_TIMESTAMP() - 5184000)),
('tag-0001-0000-0000-000000000009', '热血', 'mood', 32, FROM_UNIXTIME(UNIX_TIMESTAMP() - 5184000)),
('tag-0001-0000-0000-000000000010', '动漫风', 'style', 67, FROM_UNIXTIME(UNIX_TIMESTAMP() - 4320000));

-- ============================================================
-- 14. Style Configs (风格配置)
-- ============================================================

INSERT INTO style_configs (id, style, description, sample_image_url, user_id, created_at, updated_at) VALUES
(UUID(), 'anime_vibrant', '动漫风格 - 鲜艳色彩，适合奇幻冒险类故事', 'https://picsum.photos/seed/style-anime/400/300', NULL, UNIX_TIMESTAMP() - 8640000, UNIX_TIMESTAMP() - 8640000),
(UUID(), 'realistic_cinematic', '写实风格 - 电影质感，适合严肃题材', 'https://picsum.photos/seed/style-realistic/400/300', NULL, UNIX_TIMESTAMP() - 8640000, UNIX_TIMESTAMP() - 8640000),
(UUID(), 'watercolor_soft', '水彩风格 - 柔和色调，适合治愈系故事', 'https://picsum.photos/seed/style-watercolor/400/300', NULL, UNIX_TIMESTAMP() - 8640000, UNIX_TIMESTAMP() - 8640000),
(UUID(), 'noir_dark', '黑色电影风格 - 暗调，适合悬疑推理', 'https://picsum.photos/seed/style-noir/400/300', NULL, UNIX_TIMESTAMP() - 8640000, UNIX_TIMESTAMP() - 8640000);

-- ============================================================
-- 15. AI Generation Records (AI生成记录)
-- ============================================================

INSERT INTO ai_generation_records (id, user_id, type, provider, model, original_prompt, enhanced_prompt, input_tokens, output_tokens, total_tokens, status, progress, related_entity_id, related_entity_type, duration_ms, created_at, completed_at) VALUES
(UUID(), 'user-0001-0000-0000-000000000001', 'text', 'gemini', 'gemini-2.0-flash', '写一个奇幻故事的开场', '请创作一个奇幻风格的故事开场，包含主角登场和世界观铺垫...', 150, 800, 950, 'completed', 100, 'board-0001-0000-0000-000000000001', 'storyboard', 3500, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600)),
(UUID(), 'user-0001-0000-0000-000000000001', 'image', 'gemini', 'gemini-2.0-flash', '边境小镇，清晨，奇幻风格', '奇幻风格的边境小镇场景，清晨阳光，远处山脉...', 80, 0, 80, 'completed', 100, 'sbs-0001-0000-0000-000000000001', 'storyboard_scene', 5200, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600)),
(UUID(), 'user-0001-0000-0000-000000000001', 'text', 'gemini', 'gemini-2.0-flash', '精灵少女与冒险者初次相遇', '创作一个精灵少女与人类冒险者初次相遇的场景...', 200, 1200, 1400, 'completed', 100, 'board-0001-0000-0000-000000000002', 'storyboard', 4200, FROM_UNIXTIME(UNIX_TIMESTAMP() - 1987200), FROM_UNIXTIME(UNIX_TIMESTAMP() - 1987200));

-- ============================================================
-- 16. View History (浏览历史)
-- ============================================================

INSERT INTO view_histories (id, user_id, entity_type, entity_id, duration, viewed_at) VALUES
(UUID(), 'user-0001-0000-0000-000000000002', 'story', 'story-0001-0000-0000-000000000001', 180, FROM_UNIXTIME(UNIX_TIMESTAMP() - 777600)),
(UUID(), 'user-0001-0000-0000-000000000002', 'storyboard', 'board-0001-0000-0000-000000000001', 120, FROM_UNIXTIME(UNIX_TIMESTAMP() - 777600)),
(UUID(), 'user-0001-0000-0000-000000000002', 'story', 'story-0001-0000-0000-000000000002', 240, FROM_UNIXTIME(UNIX_TIMESTAMP() - 604800)),
(UUID(), 'user-0001-0000-0000-000000000003', 'story', 'story-0001-0000-0000-000000000001', 300, FROM_UNIXTIME(UNIX_TIMESTAMP() - 1296000)),
(UUID(), 'user-0001-0000-0000-000000000004', 'story', 'story-0001-0000-0000-000000000001', 90, FROM_UNIXTIME(UNIX_TIMESTAMP() - 172800));

-- ============================================================
-- 17. Search History (搜索历史)
-- ============================================================

INSERT INTO search_histories (id, user_id, query, type, result_count, created_at) VALUES
(UUID(), 'user-0001-0000-0000-000000000002', '龙之谷', 'story', 3, FROM_UNIXTIME(UNIX_TIMESTAMP() - 777600)),
(UUID(), 'user-0001-0000-0000-000000000002', '科幻', 'story', 5, FROM_UNIXTIME(UNIX_TIMESTAMP() - 604800)),
(UUID(), 'user-0001-0000-0000-000000000003', '精灵', 'character', 2, FROM_UNIXTIME(UNIX_TIMESTAMP() - 1036800)),
(UUID(), 'user-0001-0000-0000-000000000004', 'Alice', 'user', 1, FROM_UNIXTIME(UNIX_TIMESTAMP() - 259200));

-- ============================================================
-- 18. Invitation Codes (邀请码)
-- ============================================================

INSERT INTO invitation_codes (id, code, created_by, used_by, used_at, is_active, max_uses, current_uses, expires_at, description, created_at, updated_at) VALUES
(UUID(), 'WELCOME2024', 'user-0001-0000-0000-000000000001', NULL, NULL, 1, 100, 15, FROM_UNIXTIME(UNIX_TIMESTAMP() + 31536000), '新用户欢迎邀请码', FROM_UNIXTIME(UNIX_TIMESTAMP() - 2592000), FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400)),
(UUID(), 'VIPTEST', 'user-0001-0000-0000-000000000005', 'user-0001-0000-0000-000000000004', FROM_UNIXTIME(UNIX_TIMESTAMP() - 172800), 1, 10, 1, FROM_UNIXTIME(UNIX_TIMESTAMP() + 15552000), 'VIP用户测试邀请码', FROM_UNIXTIME(UNIX_TIMESTAMP() - 864000), FROM_UNIXTIME(UNIX_TIMESTAMP() - 172800));

-- ============================================================
-- 19. User Login Records (登录记录)
-- ============================================================

INSERT INTO user_login_records (user_id, ip_address, location, device, os, browser, user_agent, login_at, created_at) VALUES
('user-0001-0000-0000-000000000001', '192.168.1.100', '上海', 'iPhone', 'iOS 17.0', 'Safari', 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)', FROM_UNIXTIME(UNIX_TIMESTAMP() - 3600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 3600)),
('user-0001-0000-0000-000000000001', '192.168.1.101', '上海', 'MacBook Pro', 'macOS 14.0', 'Chrome', 'Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0)', FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400), FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400)),
('user-0001-0000-0000-000000000002', '192.168.1.102', '北京', 'Windows PC', 'Windows 11', 'Chrome', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)', FROM_UNIXTIME(UNIX_TIMESTAMP() - 7200), FROM_UNIXTIME(UNIX_TIMESTAMP() - 7200)),
('user-0001-0000-0000-000000000004', '192.168.1.103', '杭州', 'Android Phone', 'Android 14', 'Chrome Mobile', 'Mozilla/5.0 (Linux; Android 14)', FROM_UNIXTIME(UNIX_TIMESTAMP() - 172800), FROM_UNIXTIME(UNIX_TIMESTAMP() - 172800));

-- ============================================================
-- 20. Assets (资产)
-- ============================================================

INSERT INTO assets (id, user_id, type, name, url, thumbnail, size, mime_type, width, height, tags, usage_count, created_at) VALUES
(UUID(), 'user-0001-0000-0000-000000000001', 'image', '龙之谷_封面设计.png', 'https://picsum.photos/seed/asset1/800/1200', 'https://picsum.photos/seed/asset1-thumb/200/300', 524288, 'image/png', 800, 1200, '["封面","龙之谷","设计"]', 3, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2160000)),
(UUID(), 'user-0001-0000-0000-000000000001', 'image', '艾瑞克_角色设定.png', 'https://picsum.photos/seed/asset2/600/800', 'https://picsum.photos/seed/asset2-thumb/150/200', 314572, 'image/png', 600, 800, '["角色","艾瑞克","设定"]', 5, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600)),
(UUID(), 'user-0001-0000-0000-000000000005', 'image', '咖啡馆_场景参考.jpg', 'https://picsum.photos/seed/asset3/1920/1080', 'https://picsum.photos/seed/asset3-thumb/320/180', 1048576, 'image/jpeg', 1920, 1080, '["场景","咖啡馆","参考"]', 2, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2505600));

-- ============================================================
-- 21. Story Contributors (故事贡献者)
-- ============================================================

INSERT INTO story_contributors (id, story_id, user_id, role, invited_by, joined_at) VALUES
(UUID(), 'story-0001-0000-0000-000000000001', 'user-0001-0000-0000-000000000001', 'owner', NULL, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2160000)),
(UUID(), 'story-0001-0000-0000-000000000001', 'user-0001-0000-0000-000000000003', 'contributor', 'user-0001-0000-0000-000000000001', FROM_UNIXTIME(UNIX_TIMESTAMP() - 1728000)),
(UUID(), 'story-0001-0000-0000-000000000002', 'user-0001-0000-0000-000000000001', 'owner', NULL, FROM_UNIXTIME(UNIX_TIMESTAMP() - 1728000)),
(UUID(), 'story-0001-0000-0000-000000000003', 'user-0001-0000-0000-000000000003', 'owner', NULL, FROM_UNIXTIME(UNIX_TIMESTAMP() - 1209600)),
(UUID(), 'story-0001-0000-0000-000000000005', 'user-0001-0000-0000-000000000005', 'owner', NULL, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2505600));

-- ============================================================
-- 22. Storyboard Scene Links (故事板-场景关联)
-- ============================================================

INSERT INTO storyboard_scene_links (id, storyboard_id, story_scene_id, sequence, is_primary_scene, created_at, updated_at) VALUES
(UUID(), 'board-0001-0000-0000-000000000001', 'scene-0001-0000-0000-000000000001', 0, 1, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600)),
(UUID(), 'board-0001-0000-0000-000000000001', 'scene-0001-0000-0000-000000000002', 1, 0, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600)),
(UUID(), 'board-0001-0000-0000-000000000002', 'scene-0001-0000-0000-000000000002', 0, 1, FROM_UNIXTIME(UNIX_TIMESTAMP() - 1987200), FROM_UNIXTIME(UNIX_TIMESTAMP() - 1987200)),
(UUID(), 'board-0001-0000-0000-000000000002', 'scene-0001-0000-0000-000000000003', 1, 0, FROM_UNIXTIME(UNIX_TIMESTAMP() - 1987200), FROM_UNIXTIME(UNIX_TIMESTAMP() - 1987200)),
(UUID(), 'board-0001-0000-0000-000000000004', 'scene-0001-0000-0000-000000000004', 0, 1, FROM_UNIXTIME(UNIX_TIMESTAMP() - 1641600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 1641600)),
(UUID(), 'board-0001-0000-0000-000000000004', 'scene-0001-0000-0000-000000000005', 1, 0, FROM_UNIXTIME(UNIX_TIMESTAMP() - 1641600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 1641600));

-- ============================================================
-- 23. Storyboard Character Links (故事板-角色关联)
-- ============================================================

INSERT INTO storyboard_character_links (id, storyboard_id, character_id, role, ordering, notes, created_at, updated_at) VALUES
(UUID(), 'board-0001-0000-0000-000000000001', 'char-0001-0000-0000-000000000001', '主角', 0, '艾瑞克', FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600)),
(UUID(), 'board-0001-0000-0000-000000000002', 'char-0001-0000-0000-000000000001', '主角', 0, '艾瑞克', FROM_UNIXTIME(UNIX_TIMESTAMP() - 1987200), FROM_UNIXTIME(UNIX_TIMESTAMP() - 1987200)),
(UUID(), 'board-0001-0000-0000-000000000002', 'char-0001-0000-0000-000000000002', '女主角', 1, '莉娜', FROM_UNIXTIME(UNIX_TIMESTAMP() - 1987200), FROM_UNIXTIME(UNIX_TIMESTAMP() - 1987200)),
(UUID(), 'board-0001-0000-0000-000000000003', 'char-0001-0000-0000-000000000001', '主角', 0, '艾瑞克-商队线', FROM_UNIXTIME(UNIX_TIMESTAMP() - 1728000), FROM_UNIXTIME(UNIX_TIMESTAMP() - 1728000)),
(UUID(), 'board-0001-0000-0000-000000000004', 'char-0001-0000-0000-000000000003', '主角', 0, '陈星河舰长', FROM_UNIXTIME(UNIX_TIMESTAMP() - 1641600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 1641600));

-- ============================================================
-- 24. Story Tags (故事-标签关联)
-- ============================================================

INSERT INTO story_tags (id, story_id, tag_id, created_at) VALUES
(UUID(), 'story-0001-0000-0000-000000000001', 'tag-0001-0000-0000-000000000001', FROM_UNIXTIME(UNIX_TIMESTAMP() - 2160000)),
(UUID(), 'story-0001-0000-0000-000000000001', 'tag-0001-0000-0000-000000000005', FROM_UNIXTIME(UNIX_TIMESTAMP() - 2160000)),
(UUID(), 'story-0001-0000-0000-000000000001', 'tag-0001-0000-0000-000000000009', FROM_UNIXTIME(UNIX_TIMESTAMP() - 2160000)),
(UUID(), 'story-0001-0000-0000-000000000002', 'tag-0001-0000-0000-000000000002', FROM_UNIXTIME(UNIX_TIMESTAMP() - 1728000)),
(UUID(), 'story-0001-0000-0000-000000000002', 'tag-0001-0000-0000-000000000007', FROM_UNIXTIME(UNIX_TIMESTAMP() - 1728000)),
(UUID(), 'story-0001-0000-0000-000000000003', 'tag-0001-0000-0000-000000000003', FROM_UNIXTIME(UNIX_TIMESTAMP() - 1209600)),
(UUID(), 'story-0001-0000-0000-000000000005', 'tag-0001-0000-0000-000000000004', FROM_UNIXTIME(UNIX_TIMESTAMP() - 2505600)),
(UUID(), 'story-0001-0000-0000-000000000005', 'tag-0001-0000-0000-000000000008', FROM_UNIXTIME(UNIX_TIMESTAMP() - 2505600));

-- ============================================================
-- 25. Character Tags (角色-标签关联)
-- ============================================================

INSERT INTO character_tags (id, character_id, tag_id, created_at) VALUES
(UUID(), 'char-0001-0000-0000-000000000001', 'tag-0001-0000-0000-000000000009', FROM_UNIXTIME(UNIX_TIMESTAMP() - 2160000)),
(UUID(), 'char-0001-0000-0000-000000000002', 'tag-0001-0000-0000-000000000001', FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600)),
(UUID(), 'char-0001-0000-0000-000000000003', 'tag-0001-0000-0000-000000000002', FROM_UNIXTIME(UNIX_TIMESTAMP() - 1728000)),
(UUID(), 'char-0001-0000-0000-000000000004', 'tag-0001-0000-0000-000000000003', FROM_UNIXTIME(UNIX_TIMESTAMP() - 1209600)),
(UUID(), 'char-0001-0000-0000-000000000005', 'tag-0001-0000-0000-000000000008', FROM_UNIXTIME(UNIX_TIMESTAMP() - 2505600));

-- ============================================================
-- 26. User Follows (用户关注关系 - 专用表)
-- ============================================================

INSERT INTO user_follows (id, follower_id, followee_id, created_at) VALUES
(UUID(), 'user-0001-0000-0000-000000000002', 'user-0001-0000-0000-000000000001', FROM_UNIXTIME(UNIX_TIMESTAMP() - 864000)),
(UUID(), 'user-0001-0000-0000-000000000002', 'user-0001-0000-0000-000000000003', FROM_UNIXTIME(UNIX_TIMESTAMP() - 691200)),
(UUID(), 'user-0001-0000-0000-000000000003', 'user-0001-0000-0000-000000000001', FROM_UNIXTIME(UNIX_TIMESTAMP() - 1296000)),
(UUID(), 'user-0001-0000-0000-000000000004', 'user-0001-0000-0000-000000000001', FROM_UNIXTIME(UNIX_TIMESTAMP() - 172800)),
(UUID(), 'user-0001-0000-0000-000000000004', 'user-0001-0000-0000-000000000005', FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400)),
(UUID(), 'user-0001-0000-0000-000000000005', 'user-0001-0000-0000-000000000001', FROM_UNIXTIME(UNIX_TIMESTAMP() - 1728000)),
(UUID(), 'user-0001-0000-0000-000000000005', 'user-0001-0000-0000-000000000003', FROM_UNIXTIME(UNIX_TIMESTAMP() - 1555200));

-- ============================================================
-- 27. Story Likes (故事点赞 - 专用表)
-- ============================================================

INSERT INTO story_likes (id, user_id, story_id, created_at) VALUES
(UUID(), 'user-0001-0000-0000-000000000002', 'story-0001-0000-0000-000000000001', FROM_UNIXTIME(UNIX_TIMESTAMP() - 777600)),
(UUID(), 'user-0001-0000-0000-000000000002', 'story-0001-0000-0000-000000000002', FROM_UNIXTIME(UNIX_TIMESTAMP() - 604800)),
(UUID(), 'user-0001-0000-0000-000000000003', 'story-0001-0000-0000-000000000001', FROM_UNIXTIME(UNIX_TIMESTAMP() - 1296000)),
(UUID(), 'user-0001-0000-0000-000000000004', 'story-0001-0000-0000-000000000001', FROM_UNIXTIME(UNIX_TIMESTAMP() - 172800)),
(UUID(), 'user-0001-0000-0000-000000000005', 'story-0001-0000-0000-000000000001', FROM_UNIXTIME(UNIX_TIMESTAMP() - 1728000)),
(UUID(), 'user-0001-0000-0000-000000000005', 'story-0001-0000-0000-000000000002', FROM_UNIXTIME(UNIX_TIMESTAMP() - 1555200));

-- ============================================================
-- 28. Story Follows (故事关注)
-- ============================================================

INSERT INTO story_follows (id, user_id, story_id, created_at) VALUES
(UUID(), 'user-0001-0000-0000-000000000002', 'story-0001-0000-0000-000000000001', FROM_UNIXTIME(UNIX_TIMESTAMP() - 777600)),
(UUID(), 'user-0001-0000-0000-000000000003', 'story-0001-0000-0000-000000000001', FROM_UNIXTIME(UNIX_TIMESTAMP() - 1296000)),
(UUID(), 'user-0001-0000-0000-000000000002', 'story-0001-0000-0000-000000000002', FROM_UNIXTIME(UNIX_TIMESTAMP() - 604800));

-- ============================================================
-- 29. Character Follows (角色关注)
-- ============================================================

INSERT INTO character_follows (id, user_id, character_id, created_at) VALUES
(UUID(), 'user-0001-0000-0000-000000000002', 'char-0001-0000-0000-000000000001', FROM_UNIXTIME(UNIX_TIMESTAMP() - 777600)),
(UUID(), 'user-0001-0000-0000-000000000002', 'char-0001-0000-0000-000000000002', FROM_UNIXTIME(UNIX_TIMESTAMP() - 691200)),
(UUID(), 'user-0001-0000-0000-000000000005', 'char-0001-0000-0000-000000000005', FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400));

-- ============================================================
-- 30. Memberships (会员信息)
-- ============================================================

INSERT INTO memberships (id, user_id, tier, status, start_date, end_date, auto_renew, token_quota, token_used, storage_quota, storage_used, created_at, updated_at) VALUES
(UUID(), 'user-0001-0000-0000-000000000005', 'pro', 'active', FROM_UNIXTIME(UNIX_TIMESTAMP() - 5184000), FROM_UNIXTIME(UNIX_TIMESTAMP() + 2592000), 1, 100000, 15000, 10737418240, 2147483648, FROM_UNIXTIME(UNIX_TIMESTAMP() - 5184000), FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400)),
(UUID(), 'user-0001-0000-0000-000000000001', 'basic', 'active', FROM_UNIXTIME(UNIX_TIMESTAMP() - 2592000), FROM_UNIXTIME(UNIX_TIMESTAMP() + 86400), 0, 10000, 2500, 1073741824, 524288000, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2592000), FROM_UNIXTIME(UNIX_TIMESTAMP() - 3600));

-- ============================================================
-- 31. User Referrals (用户邀请记录)
-- ============================================================

INSERT INTO user_referrals (id, referrer_id, referee_id, referral_code, points_earned, status, created_at, rewarded_at) VALUES
(UUID(), 'user-0001-0000-0000-000000000005', 'user-0001-0000-0000-000000000004', 'VIPTEST', 500, 'completed', FROM_UNIXTIME(UNIX_TIMESTAMP() - 172800), FROM_UNIXTIME(UNIX_TIMESTAMP() - 172800));

-- ============================================================
-- 32. Third Party Logins (第三方登录)
-- ============================================================

INSERT INTO third_party_logins (id, user_id, provider, provider_user_id, provider_email, provider_user_name, status, created_at, updated_at) VALUES
(UUID(), 'user-0001-0000-0000-000000000001', 'google', 'google-alice-123', 'alice@test.com', 'Alice 创作者', 1, UNIX_TIMESTAMP() - 2592000, UNIX_TIMESTAMP() - 3600),
(UUID(), 'user-0001-0000-0000-000000000002', 'apple', 'apple-bob-456', 'bob@test.com', 'Bob 读者', 1, UNIX_TIMESTAMP() - 1728000, UNIX_TIMESTAMP() - 7200),
(UUID(), 'user-0001-0000-0000-000000000005', 'google', 'google-eve-789', 'eve@test.com', 'Eve VIP', 1, UNIX_TIMESTAMP() - 5184000, UNIX_TIMESTAMP() - 600);

-- ============================================================
-- 33. Fragment Comments (碎片评论)
-- ============================================================

INSERT INTO fragment_comments (id, fragment_id, user_id, content, parent_id, created_at, updated_at) VALUES
(UUID(), 'frag-0001-0000-0000-000000000001', 'user-0001-0000-0000-000000000002', '这个时间倒流的设定太有意思了！期待看到完整故事', NULL, UNIX_TIMESTAMP() - 864000, UNIX_TIMESTAMP() - 864000),
(UUID(), 'frag-0001-0000-0000-000000000001', 'user-0001-0000-0000-000000000003', '同感！一分钟倒流遇到另一个倒流的人，这个设定很有张力', NULL, UNIX_TIMESTAMP() - 860000, UNIX_TIMESTAMP() - 860000),
(UUID(), 'frag-0001-0000-0000-000000000005', 'user-0001-0000-0000-000000000005', '欢迎 Diana！创作是一个循序渐进的过程，加油！', NULL, UNIX_TIMESTAMP() - 172800, UNIX_TIMESTAMP() - 172800);

-- ============================================================
-- 34. Fragment Likes (碎片点赞)
-- ============================================================

INSERT INTO fragment_likes (id, fragment_id, user_id, created_at) VALUES
(UUID(), 'frag-0001-0000-0000-000000000001', 'user-0001-0000-0000-000000000002', UNIX_TIMESTAMP() - 864000),
(UUID(), 'frag-0001-0000-0000-000000000001', 'user-0001-0000-0000-000000000003', UNIX_TIMESTAMP() - 860000),
(UUID(), 'frag-0001-0000-0000-000000000002', 'user-0001-0000-0000-000000000003', UNIX_TIMESTAMP() - 691200),
(UUID(), 'frag-0001-0000-0000-000000000004', 'user-0001-0000-0000-000000000005', UNIX_TIMESTAMP() - 604800);

-- ============================================================
-- 35. Fragment Shares (碎片分享)
-- ============================================================

INSERT INTO fragment_shares (id, fragment_id, user_id, platform, created_at) VALUES
(UUID(), 'frag-0001-0000-0000-000000000001', 'user-0001-0000-0000-000000000002', 'wechat', UNIX_TIMESTAMP() - 864000),
(UUID(), 'frag-0001-0000-0000-000000000002', 'user-0001-0000-0000-000000000003', 'twitter', UNIX_TIMESTAMP() - 691200);

-- ============================================================
-- 36. Agents (角色 AI Agent)
-- ============================================================

INSERT INTO agents (id, character_id, name, description, status, system_prompt, temperature, provider, model, max_tokens, interaction_count, skill_count, created_at, updated_at) VALUES
(UUID(), 'char-0001-0000-0000-000000000001', '艾瑞克 Agent', '龙之谷传说中的年轻冒险者，勇敢善良，对世界充满好奇', 'active', '你是艾瑞克，龙之谷传说中的年轻冒险者。你勇敢、善良、正义感强，对历史和传说特别感兴趣。', 0.7, 'gemini', 'gemini-2.0-flash', 2048, 156, 3, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 432000)),
(UUID(), 'char-0001-0000-0000-000000000002', '莉娜 Agent', '神秘的精灵少女，守护精灵森林，对外界充满好奇', 'active', '你是莉娜，精灵族的后裔，守护着精灵森林。你神秘、谨慎、善良，对人类世界知之甚少但充满好奇。', 0.7, 'gemini', 'gemini-2.0-flash', 2048, 89, 2, FROM_UNIXTIME(UNIX_TIMESTAMP() - 1987200), FROM_UNIXTIME(UNIX_TIMESTAMP() - 432000)),
(UUID(), 'char-0001-0000-0000-000000000005', '苏晓雨 Agent', '年轻设计师，活泼善良，追求自由和梦想', 'active', '你是苏晓雨，年轻的设计师。你活泼、善良、独立，热爱生活和艺术，正在为自己的工作室努力。', 0.8, 'gemini', 'gemini-2.0-flash', 2048, 45, 2, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2505600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400));

-- ============================================================
-- 37. Panels (故事分章/面板)
-- ============================================================

INSERT INTO panels (id, story_id, sequence, title, content, image, likes, published, created_at, updated_at) VALUES
('panel-0001-0000-0000-000000000001', 'story-0001-0000-0000-000000000001', 1, '序章', '在边境小镇的山丘上，艾瑞克眺望着远方被云雾笼罩的山脉...', 'https://picsum.photos/seed/panel1/800/450', 45, 1, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2160000), FROM_UNIXTIME(UNIX_TIMESTAMP() - 432000)),
('panel-0001-0000-0000-000000000002', 'story-0001-0000-0000-000000000001', 2, '精灵森林', '穿越茂密的森林，艾瑞克感到有目光在注视着自己...', 'https://picsum.photos/seed/panel2/800/450', 34, 1, FROM_UNIXTIME(UNIX_TIMESTAMP() - 2073600), FROM_UNIXTIME(UNIX_TIMESTAMP() - 432000));

-- ============================================================
-- 38. 更多 Stories (扩展故事数据)
-- ============================================================

INSERT INTO stories (id, title, description, cover_image, author_id, likes, followers, saves, panels, storyboard_count, default_scene_count, genre, style, status, is_collaboration_open, visibility, use_ai, ai_enabled, created_at, updated_at) VALUES
-- 故事 5: 都市治愈
('story-0001-0000-0000-000000000006', '雨天的书店', '一个下雨的午后，失意的作家走进了一家旧书店。店主是一位神秘的老人，他递给作家一本没有字的书。当作家翻开书页，故事开始了...', 'https://picsum.photos/seed/rainy-bookstore/400/600', 'user-0001-0000-0000-000000000003', 423, 89, 67, 0, 2, 3, 'slice_of_life', '{"artStyle":"watercolor","colorPalette":"muted","lighting":"soft"}', 'published', 0, 'public', 1, 1, FROM_UNIXTIME(UNIX_TIMESTAMP() - 864000), FROM_UNIXTIME(UNIX_TIMESTAMP() - 43200));

-- ============================================================
-- 39. 更多 Fragments (扩展碎片) - 丰富内容供碎片 Feed 和详情展示
-- ============================================================

INSERT INTO fragments (id, creator_id, content, image_urls, visibility, source_type, is_converted, likes, comments, shares, views, created_at, updated_at) VALUES
('frag-0001-0000-0000-000000000006', 'user-0001-0000-0000-000000000001', '【世界观设定】在一个魔法与科技并存的时代，人们通过「记忆晶石」可以回溯并修改自己的过去。但每次修改都会产生「蝴蝶效应」，改变的不只是自己的人生，还有整个世界的走向。主角在一次意外中获得了晶石，却发现自己每次回溯都会失去一段记忆...', '["https://picsum.photos/seed/frag6/400/300"]', 'public', 'original', 0, 67, 18, 4, 890, UNIX_TIMESTAMP() - 518400, UNIX_TIMESTAMP() - 518400),
('frag-0001-0000-0000-000000000007', 'user-0001-0000-0000-000000000005', '分享《咖啡馆邂逅》里最喜欢的一段：苏晓雨和陌生人的对话从一本书开始，那种自然而然的默契太美了。有时候最好的相遇，就是不需要刻意安排的。', '["https://picsum.photos/seed/frag7/400/300"]', 'public', 'story_excerpt', 0, 123, 28, 15, 1567, UNIX_TIMESTAMP() - 345600, UNIX_TIMESTAMP() - 345600),
('frag-0001-0000-0000-000000000008', 'user-0001-0000-0000-000000000001', '【开篇灵感】雨夜，便利店。一个穿着湿透外套的男人推门进来，在货架前站了很久。店员注意到他盯着同一包泡面已经十分钟了。「需要帮忙吗？」男人转过头，眼神空洞：「你说，如果当初我选了另一包，人生会不会不一样？」', '["https://picsum.photos/seed/frag8/400/300"]', 'public', 'original', 0, 234, 56, 23, 3456, UNIX_TIMESTAMP() - 259200, UNIX_TIMESTAMP() - 259200),
('frag-0001-0000-0000-000000000009', 'user-0001-0000-0000-000000000003', '【对话片段】「你为什么总是看着窗外？」「因为那里有答案。」「什么答案？」「关于我为什么还活着的答案。」她沉默了很久，然后说：「那你看够了吗？」他笑了：「没有。但我可以一边看，一边陪你说话。」', '["https://picsum.photos/seed/frag9/400/300"]', 'public', 'original', 0, 178, 42, 18, 2345, UNIX_TIMESTAMP() - 172800, UNIX_TIMESTAMP() - 172800),
('frag-0001-0000-0000-000000000010', 'user-0001-0000-0000-000000000005', '【场景描写】午后的阳光透过落地窗洒在木质地板上，空气中飘着咖啡和旧书的味道。她坐在靠窗的位置，手指轻轻划过书页。窗外是熙攘的街道，窗内是静止的时光。也许这就是她喜欢这里的原因——在喧嚣中偷一点安静。', '["https://picsum.photos/seed/frag10/400/300","https://picsum.photos/seed/frag10b/400/300"]', 'public', 'original', 0, 312, 89, 34, 4567, UNIX_TIMESTAMP() - 86400, UNIX_TIMESTAMP() - 86400),
('frag-0001-0000-0000-000000000011', 'user-0001-0000-0000-000000000002', '从《星际迷航：新纪元》摘录：陈星河舰长站在舰桥上，全息屏幕上的数据流不断闪烁。「舰长，这个信号...它来自一个从未被记录的星系。」科学官的声音有些颤抖，「而且...它似乎是故意发送给我们的。」全舰一片死寂。', '["https://picsum.photos/seed/frag11/400/300"]', 'public', 'story_excerpt', 0, 89, 23, 9, 1234, UNIX_TIMESTAMP() - 43200, UNIX_TIMESTAMP() - 43200),
('frag-0001-0000-0000-000000000012', 'user-0001-0000-0000-000000000001', '【角色灵感】一个能听见别人心声的侦探，却听不见自己的心。他破案无数，却始终无法解开自己童年的谜题。直到有一天，他遇到了一个「没有心声」的人——要么是死人，要么是...来自另一个世界。', '["https://picsum.photos/seed/frag12/400/300"]', 'public', 'original', 0, 445, 112, 45, 5678, UNIX_TIMESTAMP() - 3600, UNIX_TIMESTAMP() - 3600),
('frag-0001-0000-0000-000000000013', 'user-0001-0000-0000-000000000004', '【练习】尝试写一个反转：表面上是爱情故事，最后发现是悬疑。男女主相遇、相爱、结婚，直到某天女主发现丈夫的日记里写着「任务完成，目标已清除」。而她的名字，就在目标名单上。', '["https://picsum.photos/seed/frag13/400/300"]', 'public', 'original', 0, 156, 34, 12, 1890, UNIX_TIMESTAMP() - 7200, UNIX_TIMESTAMP() - 7200),
('frag-0001-0000-0000-000000000014', 'user-0001-0000-0000-000000000003', '【雨天的书店】灵感记录：当作家翻开那本无字之书，第一页开始浮现文字。不是印刷体，而是手写——和他自己的笔迹一模一样。最后一页写着：你终于来了。我等你很久了。署名是三十年前的日期。', '["https://picsum.photos/seed/frag14/400/300"]', 'public', 'original', 0, 267, 67, 28, 3456, UNIX_TIMESTAMP() - 14400, UNIX_TIMESTAMP() - 14400),
('frag-0001-0000-0000-000000000015', 'user-0001-0000-0000-000000000005', '深夜食堂式的设定：一家只在凌晨 0 点到 4 点营业的面馆。来的都是睡不着的人。老板从不问你的故事，但总能在你吃完那碗面之后，让你觉得...今晚可以睡个好觉了。', '["https://picsum.photos/seed/frag15/400/300"]', 'public', 'original', 0, 534, 145, 67, 7890, UNIX_TIMESTAMP() - 1800, UNIX_TIMESTAMP() - 1800);

-- ============================================================
-- Done! Mock data inserted successfully.
-- ============================================================

SELECT 'Mock data inserted successfully!' AS status;
SELECT
    (SELECT COUNT(*) FROM users) AS users,
    (SELECT COUNT(*) FROM stories) AS stories,
    (SELECT COUNT(*) FROM storyboards) AS storyboards,
    (SELECT COUNT(*) FROM characters) AS characters,
    (SELECT COUNT(*) FROM fragments) AS fragments,
    (SELECT COUNT(*) FROM comments) AS comments,
    (SELECT COUNT(*) FROM notifications) AS notifications,
    (SELECT COUNT(*) FROM likes) AS likes;
