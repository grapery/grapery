-- ============================================================
-- King 用户 Mock 数据
-- 运行于 mock_data.sql 之后: mysql -u root -p12345678 -h 127.0.0.1 grapery < scripts/king_mock_data.sql
-- ============================================================

-- 统一 collation，避免 utf8mb4_unicode_ci 与 utf8mb4_0900_ai_ci 混用报错
SET NAMES utf8mb4 COLLATE utf8mb4_0900_ai_ci;

SET @king_id = '6492a794-5c56-40a6-9979-b5aeafc4b2f3';
SET @ts = 1772883780;
SET @ts_ms = 1772883780000;

-- 1. King 用户
INSERT INTO users (id, username, email, password_hash, display_name, avatar, background, bio, location, website, followers, following, storyboard_count, fragments_count, status, email_verified, last_login_at, points, referral_code, created_at, updated_at) VALUES
(@king_id, 'King', '435594427@qq.com', '$2a$10$vQgv55z4bNt3.tQZyGbFmuSYSwyEkIdsPH5o184jkH0qE2L1u6.XC', 'King', 'https://picsum.photos/seed/alice/200/200', 'https://picsum.photos/seed/alice-bg/800/400', '热爱创作奇幻故事，专注于角色塑造和世界观构建', '上海', 'https://alice-portfolio.com', 5, 5, 20, 100, 'active', 1, @ts_ms, 100, 'KING2024', FROM_UNIXTIME(@ts), FROM_UNIXTIME(@ts))
ON DUPLICATE KEY UPDATE updated_at = FROM_UNIXTIME(@ts);

-- 2. King 用户设置
INSERT IGNORE INTO user_settings (id, user_id, language, theme, font_size, data_saver, profile_visibility, default_story_visibility, default_fragment_visibility, allow_follow_from, allow_comments_from, allow_messages_from, show_online_status, show_read_receipts, ai_enabled, a_idata_sharing, notification_settings, updated_at)
SELECT UUID(), @king_id, 'zh-CN', 'dark', 'medium', 0, 'public', 'public', 'public', 'everyone', 'everyone', 'followers_only', 1, 1, 1, 1, '{"push":true,"email":true,"likes":true,"comments":true,"follows":true}', UNIX_TIMESTAMP()
WHERE NOT EXISTS (SELECT 1 FROM user_settings WHERE user_id COLLATE utf8mb4_0900_ai_ci = @king_id COLLATE utf8mb4_0900_ai_ci LIMIT 1);

-- 3. King 的 3 个故事
INSERT INTO stories (id, title, description, cover_image, author_id, likes, followers, saves, panels, storyboard_count, default_scene_count, genre, style, status, is_collaboration_open, visibility, use_ai, ai_enabled, created_at, updated_at) VALUES
('story-6492-0000-0000-000000000001', '星辰之约', '在星际殖民时代，一名年轻的导航员发现了通往未知星系的坐标。她必须在一场星际战争爆发前，将秘密传递给能够改变命运的人。', 'https://picsum.photos/seed/king-story1/400/600', @king_id, 156, 89, 45, 0, 7, 4, 'scifi', '{"artStyle":"realistic","colorPalette":"cool","lighting":"cinematic"}', 'published', 0, 'public', 1, 1, FROM_UNIXTIME(@ts - 86400), FROM_UNIXTIME(@ts)),
('story-6492-0000-0000-000000000002', '迷雾森林', '传说中，迷雾森林的深处藏着能实现愿望的泉水。但每个踏入森林的人，都会在雾中迷失方向，只有真正纯粹的心才能找到出路。', 'https://picsum.photos/seed/king-story2/400/600', @king_id, 234, 123, 67, 0, 7, 3, 'fantasy', '{"artStyle":"anime","colorPalette":"muted","lighting":"mysterious"}', 'published', 0, 'public', 1, 1, FROM_UNIXTIME(@ts - 43200), FROM_UNIXTIME(@ts)),
('story-6492-0000-0000-000000000003', '午夜电台', '一家只在午夜播出的电台，接听来自失眠者的电话。主持人从不透露自己的身份，但每个听过节目的人，都会在第二天做出一个改变人生的决定。', 'https://picsum.photos/seed/king-story3/400/600', @king_id, 189, 78, 34, 0, 6, 3, 'mystery', '{"artStyle":"noir","colorPalette":"dark","lighting":"shadowy"}', 'published', 0, 'public', 1, 1, FROM_UNIXTIME(@ts - 21600), FROM_UNIXTIME(@ts));

-- 4. King 故事的贡献者
INSERT INTO story_contributors (id, story_id, user_id, role, invited_by, joined_at) VALUES
(UUID(), 'story-6492-0000-0000-000000000001', @king_id, 'owner', NULL, FROM_UNIXTIME(@ts - 86400)),
(UUID(), 'story-6492-0000-0000-000000000002', @king_id, 'owner', NULL, FROM_UNIXTIME(@ts - 43200)),
(UUID(), 'story-6492-0000-0000-000000000003', @king_id, 'owner', NULL, FROM_UNIXTIME(@ts - 21600));

-- 5. King 故事的角色
INSERT INTO characters (id, story_id, name, description, avatar, author_id, personality, background, short_term_goal, long_term_goal, handling_style, cognition_range, ability_features, appearance, dress_preference, source_type, created_by, is_public, created_at, updated_at) VALUES
('char-6492-0000-0000-000000000001', 'story-6492-0000-0000-000000000001', '林星', '年轻的星际导航员，拥有罕见的空间感知能力', 'https://picsum.photos/seed/king-char1/200/200', @king_id, '["冷静","果断","正义"]', '太空学院优秀毕业生', '传递坐标', '阻止战争', '理性分析，快速决策', '对星图有天赋', '空间导航、危机处理', '黑色短发，深邃眼眸', '深蓝色导航服', 'manual', @king_id, 1, FROM_UNIXTIME(@ts - 86400), FROM_UNIXTIME(@ts)),
('char-6492-0000-0000-000000000002', 'story-6492-0000-0000-000000000002', '艾琳', '寻找愿望泉水的少女，内心纯粹', 'https://picsum.photos/seed/king-char2/200/200', @king_id, '["善良","执着","单纯"]', '村庄长大的普通女孩', '找到泉水', '实现愿望', '跟随内心指引', '对自然敏感', '与动物沟通', '棕色长发，绿色眼睛', '简朴的旅行装', 'manual', @king_id, 1, FROM_UNIXTIME(@ts - 43200), FROM_UNIXTIME(@ts)),
('char-6492-0000-0000-000000000003', 'story-6492-0000-0000-000000000003', '夜声', '午夜电台的神秘主持人', 'https://picsum.photos/seed/king-char3/200/200', @king_id, '["神秘","睿智","温柔"]', '身份成谜', '帮助来电者', '未知', '倾听与引导', '洞察人心', '共情、引导', '从未露面', '未知', 'manual', @king_id, 1, FROM_UNIXTIME(@ts - 21600), FROM_UNIXTIME(@ts));

-- 6. King 故事场景
INSERT INTO story_scenes (id, story_id, title, description, image, location, time_of_day, source_type, created_by, is_public, tags, created_at, updated_at) VALUES
('scene-6492-0000-0000-000000000001', 'story-6492-0000-0000-000000000001', '星舰舰桥', '导航中心，全息星图闪烁', 'https://picsum.photos/seed/king-scene1/800/450', '星舰', 'night', 'manual', @king_id, 1, '["科幻","太空"]', FROM_UNIXTIME(@ts - 86400), FROM_UNIXTIME(@ts)),
('scene-6492-0000-0000-000000000002', 'story-6492-0000-0000-000000000002', '森林入口', '迷雾笼罩的古老森林', 'https://picsum.photos/seed/king-scene2/800/450', '迷雾森林', 'dusk', 'manual', @king_id, 1, '["奇幻","神秘"]', FROM_UNIXTIME(@ts - 43200), FROM_UNIXTIME(@ts)),
('scene-6492-0000-0000-000000000003', 'story-6492-0000-0000-000000000003', '电台直播间', '昏暗的直播间，红色指示灯', 'https://picsum.photos/seed/king-scene3/800/450', '电台', 'night', 'manual', @king_id, 1, '["悬疑","都市"]', FROM_UNIXTIME(@ts - 21600), FROM_UNIXTIME(@ts));

-- 7. King 的 20 个故事板 (故事1: 7个, 故事2: 7个, 故事3: 6个)
INSERT INTO storyboards (id, story_id, parent_id, creator_id, title, content, raw_input, is_standalone, is_ai_generated, scene_count, workflow_status, current_step, likes, comments, shares, fork_count, views, token_consumption, created_at, updated_at) VALUES
('board-6492-0000-0000-000000000001', 'story-6492-0000-0000-000000000001', NULL, @king_id, '第一章：发现', '林星在例行检查中发现了异常坐标。她的心跳加速——这可能是改变战争走向的关键。她必须做出选择：上报，还是独自前往？', '林星发现异常坐标', 0, 1, 4, 'published', 5, 45, 12, 5, 2, 567, 1200, FROM_UNIXTIME(@ts - 82800), FROM_UNIXTIME(@ts)),
('board-6492-0000-0000-000000000002', 'story-6492-0000-0000-000000000001', 'board-6492-0000-0000-000000000001', @king_id, '第二章：启程', '林星驾驶小型侦察舰离开了母舰。星空中，她感到前所未有的孤独，也感到前所未有的坚定。', '林星独自启程', 0, 1, 3, 'published', 5, 34, 8, 3, 1, 345, 800, FROM_UNIXTIME(@ts - 79200), FROM_UNIXTIME(@ts)),
('board-6492-0000-0000-000000000003', 'story-6492-0000-0000-000000000001', 'board-6492-0000-0000-000000000001', @king_id, '第二章（分支）：留守', '林星选择上报坐标。上级派出了特遣队，而她被要求留守。透过舷窗，她看着舰队远去，心中五味杂陈。', '林星选择上报', 1, 1, 3, 'published', 5, 23, 5, 2, 0, 234, 600, FROM_UNIXTIME(@ts - 75600), FROM_UNIXTIME(@ts)),
('board-6492-0000-0000-000000000004', 'story-6492-0000-0000-000000000001', 'board-6492-0000-0000-000000000002', @king_id, '第三章：遭遇', '在跳跃点附近，林星遭遇了敌舰。一场追逐在星空中展开。', '林星遭遇敌舰', 0, 1, 4, 'published', 5, 56, 15, 7, 3, 678, 1500, FROM_UNIXTIME(@ts - 72000), FROM_UNIXTIME(@ts)),
('board-6492-0000-0000-000000000005', 'story-6492-0000-0000-000000000001', 'board-6492-0000-0000-000000000004', @king_id, '第四章：盟友', '林星被神秘舰队救下。对方声称也在寻找同样的坐标。是敌是友？', '林星遇到神秘舰队', 0, 1, 3, 'published', 5, 41, 10, 4, 1, 456, 900, FROM_UNIXTIME(@ts - 68400), FROM_UNIXTIME(@ts)),
('board-6492-0000-0000-000000000006', 'story-6492-0000-0000-000000000001', 'board-6492-0000-0000-000000000005', @king_id, '第五章：真相', '坐标指向的是一颗古老的和平信标。林星明白了——这不是武器，是希望。', '发现和平信标', 0, 1, 3, 'published', 5, 67, 18, 9, 4, 789, 1100, FROM_UNIXTIME(@ts - 64800), FROM_UNIXTIME(@ts)),
('board-6492-0000-0000-000000000007', 'story-6492-0000-0000-000000000001', 'board-6492-0000-0000-000000000006', @king_id, '终章：星辰之约', '信标被激活，全星系收到了停战信号。林星站在舰桥上，看着星空，终于露出了笑容。', '激活信标，和平降临', 0, 1, 2, 'published', 5, 89, 22, 12, 5, 890, 700, FROM_UNIXTIME(@ts - 61200), FROM_UNIXTIME(@ts)),
('board-6492-0000-0000-000000000008', 'story-6492-0000-0000-000000000002', NULL, @king_id, '第一章：入林', '艾琳站在森林入口，深吸一口气。村民们说，进去的人都没有回来。但她必须找到泉水。', '艾琳进入迷雾森林', 0, 1, 4, 'published', 5, 38, 9, 4, 1, 423, 1000, FROM_UNIXTIME(@ts - 39600), FROM_UNIXTIME(@ts)),
('board-6492-0000-0000-000000000009', 'story-6492-0000-0000-000000000002', 'board-6492-0000-0000-000000000008', @king_id, '第二章：迷途', '雾越来越浓。艾琳失去了方向，但她记得奶奶的话：跟着心走。', '艾琳在雾中迷路', 0, 1, 3, 'published', 5, 29, 6, 2, 0, 312, 750, FROM_UNIXTIME(@ts - 36000), FROM_UNIXTIME(@ts)),
('board-6492-0000-0000-000000000010', 'story-6492-0000-0000-000000000002', 'board-6492-0000-0000-000000000009', @king_id, '第三章：白鹿', '一头白鹿出现在雾中，它看了艾琳一眼，转身走去。艾琳跟了上去。', '艾琳遇到白鹿', 0, 1, 3, 'published', 5, 52, 14, 6, 2, 534, 850, FROM_UNIXTIME(@ts - 32400), FROM_UNIXTIME(@ts)),
('board-6492-0000-0000-000000000011', 'story-6492-0000-0000-000000000002', 'board-6492-0000-0000-000000000010', @king_id, '第四章：试炼', '泉水前，艾琳面临选择：实现自己的愿望，还是拯救病重的村民？', '艾琳面临试炼', 0, 1, 3, 'published', 5, 71, 19, 8, 3, 645, 950, FROM_UNIXTIME(@ts - 28800), FROM_UNIXTIME(@ts)),
('board-6492-0000-0000-000000000012', 'story-6492-0000-0000-000000000002', 'board-6492-0000-0000-000000000011', @king_id, '第五章：抉择', '艾琳将泉水带回了村庄。她的愿望？她笑了：大家的笑容就是我的愿望。', '艾琳选择拯救村民', 0, 1, 2, 'published', 5, 94, 25, 11, 4, 756, 600, FROM_UNIXTIME(@ts - 25200), FROM_UNIXTIME(@ts)),
('board-6492-0000-0000-000000000013', 'story-6492-0000-0000-000000000002', 'board-6492-0000-0000-000000000012', @king_id, '第六章：新生', '村民们康复了。艾琳站在村口，白鹿再次出现，仿佛在向她告别。', '村庄重生', 0, 1, 2, 'published', 5, 63, 16, 7, 2, 567, 500, FROM_UNIXTIME(@ts - 21600), FROM_UNIXTIME(@ts)),
('board-6492-0000-0000-000000000014', 'story-6492-0000-0000-000000000002', 'board-6492-0000-0000-000000000013', @king_id, '终章：愿望', '多年后，艾琳成为森林的守护者。她终于明白：真正的愿望，是让更多人找到自己的路。', '艾琳成为守护者', 0, 1, 2, 'published', 5, 82, 21, 9, 3, 678, 550, FROM_UNIXTIME(@ts - 18000), FROM_UNIXTIME(@ts)),
('board-6492-0000-0000-000000000015', 'story-6492-0000-0000-000000000003', NULL, @king_id, '第一章：来电', '凌晨一点，第一个电话响起。一个男人的声音颤抖着：我明天要去见一个人，可能改变一切。', '第一个午夜来电', 0, 1, 3, 'published', 5, 47, 11, 5, 2, 489, 800, FROM_UNIXTIME(@ts - 18000), FROM_UNIXTIME(@ts)),
('board-6492-0000-0000-000000000016', 'story-6492-0000-0000-000000000003', 'board-6492-0000-0000-000000000015', @king_id, '第二章：倾听', '夜声从不给建议，只是倾听。但每个挂断电话的人，眼中都有了光。', '夜声的倾听', 0, 1, 3, 'published', 5, 35, 8, 3, 1, 367, 650, FROM_UNIXTIME(@ts - 14400), FROM_UNIXTIME(@ts)),
('board-6492-0000-0000-000000000017', 'story-6492-0000-0000-000000000003', 'board-6492-0000-0000-000000000016', @king_id, '第三章：秘密', '一个女孩打来电话，说她能听见别人的想法。夜声沉默了很久。', '神秘女孩来电', 0, 1, 3, 'published', 5, 58, 15, 6, 2, 512, 900, FROM_UNIXTIME(@ts - 10800), FROM_UNIXTIME(@ts)),
('board-6492-0000-0000-000000000018', 'story-6492-0000-0000-000000000003', 'board-6492-0000-0000-000000000017', @king_id, '第四章：共鸣', '夜声说：我也曾和你一样。你不是怪物，你是被选中的。', '夜声与女孩的共鸣', 0, 1, 3, 'published', 5, 72, 20, 8, 3, 623, 850, FROM_UNIXTIME(@ts - 7200), FROM_UNIXTIME(@ts)),
('board-6492-0000-0000-000000000019', 'story-6492-0000-0000-000000000003', 'board-6492-0000-0000-000000000018', @king_id, '第五章：黎明', '节目结束，天边泛起鱼肚白。夜声关掉麦克风，摘下耳机。镜子里，没有倒影。', '夜声的秘密', 0, 1, 2, 'published', 5, 91, 24, 10, 4, 734, 600, FROM_UNIXTIME(@ts - 3600), FROM_UNIXTIME(@ts)),
('board-6492-0000-0000-000000000020', 'story-6492-0000-0000-000000000003', 'board-6492-0000-0000-000000000019', @king_id, '终章：继续', '下个午夜，电台照常播出。又一个失眠者拨通了电话。故事，还在继续。', '电台继续', 0, 1, 2, 'published', 5, 65, 17, 7, 2, 556, 500, FROM_UNIXTIME(@ts), FROM_UNIXTIME(@ts));

-- 8. 故事板场景
INSERT INTO storyboard_scenes (id, storyboard_id, story_scene_id, sequence, title, description, image, video_url, location, time_of_day, characters, mood, is_ai_generated, created_at, updated_at) VALUES
(UUID(), 'board-6492-0000-0000-000000000001', 'scene-6492-0000-0000-000000000001', 1, '发现坐标', '林星发现异常', 'https://picsum.photos/seed/king-sbs1/800/450', '', '星舰', 'night', '["林星"]', 'tense', 1, FROM_UNIXTIME(@ts - 82800), FROM_UNIXTIME(@ts)),
(UUID(), 'board-6492-0000-0000-000000000008', 'scene-6492-0000-0000-000000000002', 1, '森林入口', '艾琳踏入迷雾', 'https://picsum.photos/seed/king-sbs2/800/450', '', '迷雾森林', 'dusk', '["艾琳"]', 'mysterious', 1, FROM_UNIXTIME(@ts - 39600), FROM_UNIXTIME(@ts)),
(UUID(), 'board-6492-0000-0000-000000000015', 'scene-6492-0000-0000-000000000003', 1, '直播间', '午夜电台开播', 'https://picsum.photos/seed/king-sbs3/800/450', '', '电台', 'night', '["夜声"]', 'mysterious', 1, FROM_UNIXTIME(@ts - 18000), FROM_UNIXTIME(@ts));

-- 9. 关注关系: King 关注所有其他用户，所有其他用户关注 King
INSERT IGNORE INTO follows (id, follower_id, followable_type, followable_id, notifications_enabled, created_at) VALUES
(UUID(), @king_id, 'user', 'user-0001-0000-0000-000000000001', 1, UNIX_TIMESTAMP() - 86400),
(UUID(), @king_id, 'user', 'user-0001-0000-0000-000000000002', 1, UNIX_TIMESTAMP() - 86400),
(UUID(), @king_id, 'user', 'user-0001-0000-0000-000000000003', 1, UNIX_TIMESTAMP() - 86400),
(UUID(), @king_id, 'user', 'user-0001-0000-0000-000000000004', 1, UNIX_TIMESTAMP() - 86400),
(UUID(), @king_id, 'user', 'user-0001-0000-0000-000000000005', 1, UNIX_TIMESTAMP() - 86400),
(UUID(), 'user-0001-0000-0000-000000000001', 'user', @king_id, 1, UNIX_TIMESTAMP() - 86400),
(UUID(), 'user-0001-0000-0000-000000000002', 'user', @king_id, 1, UNIX_TIMESTAMP() - 86400),
(UUID(), 'user-0001-0000-0000-000000000003', 'user', @king_id, 1, UNIX_TIMESTAMP() - 86400),
(UUID(), 'user-0001-0000-0000-000000000004', 'user', @king_id, 1, UNIX_TIMESTAMP() - 86400),
(UUID(), 'user-0001-0000-0000-000000000005', 'user', @king_id, 1, UNIX_TIMESTAMP() - 86400);

INSERT IGNORE INTO user_follows (id, follower_id, followee_id, created_at) VALUES
(UUID(), @king_id, 'user-0001-0000-0000-000000000001', FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400)),
(UUID(), @king_id, 'user-0001-0000-0000-000000000002', FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400)),
(UUID(), @king_id, 'user-0001-0000-0000-000000000003', FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400)),
(UUID(), @king_id, 'user-0001-0000-0000-000000000004', FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400)),
(UUID(), @king_id, 'user-0001-0000-0000-000000000005', FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400)),
(UUID(), 'user-0001-0000-0000-000000000001', @king_id, FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400)),
(UUID(), 'user-0001-0000-0000-000000000002', @king_id, FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400)),
(UUID(), 'user-0001-0000-0000-000000000003', @king_id, FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400)),
(UUID(), 'user-0001-0000-0000-000000000004', @king_id, FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400)),
(UUID(), 'user-0001-0000-0000-000000000005', @king_id, FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400));

-- 10. 其他用户对 King 故事的点赞和关注
INSERT INTO story_likes (id, user_id, story_id, created_at) VALUES
(UUID(), 'user-0001-0000-0000-000000000001', 'story-6492-0000-0000-000000000001', FROM_UNIXTIME(UNIX_TIMESTAMP() - 3600)),
(UUID(), 'user-0001-0000-0000-000000000002', 'story-6492-0000-0000-000000000001', FROM_UNIXTIME(UNIX_TIMESTAMP() - 3600)),
(UUID(), 'user-0001-0000-0000-000000000003', 'story-6492-0000-0000-000000000002', FROM_UNIXTIME(UNIX_TIMESTAMP() - 3600)),
(UUID(), 'user-0001-0000-0000-000000000004', 'story-6492-0000-0000-000000000002', FROM_UNIXTIME(UNIX_TIMESTAMP() - 3600)),
(UUID(), 'user-0001-0000-0000-000000000005', 'story-6492-0000-0000-000000000003', FROM_UNIXTIME(UNIX_TIMESTAMP() - 3600));

INSERT INTO story_follows (id, user_id, story_id, created_at) VALUES
(UUID(), 'user-0001-0000-0000-000000000001', 'story-6492-0000-0000-000000000001', FROM_UNIXTIME(UNIX_TIMESTAMP() - 3600)),
(UUID(), 'user-0001-0000-0000-000000000002', 'story-6492-0000-0000-000000000001', FROM_UNIXTIME(UNIX_TIMESTAMP() - 3600)),
(UUID(), 'user-0001-0000-0000-000000000005', 'story-6492-0000-0000-000000000002', FROM_UNIXTIME(UNIX_TIMESTAMP() - 3600));

-- 11. 故事、故事板、碎片的评论
INSERT INTO comments (id, author_id, content, target_type, target_id, likes, reply_count, created_at, updated_at) VALUES
(UUID(), 'user-0001-0000-0000-000000000001', '《星辰之约》的设定太棒了！林星这个角色很有魅力', 'story', 'story-6492-0000-0000-000000000001', 12, 1, FROM_UNIXTIME(@ts - 3600), FROM_UNIXTIME(@ts - 3600)),
(UUID(), 'user-0001-0000-0000-000000000002', '分支剧情的设计很有创意，想看看留守线会怎么发展', 'storyboard', 'board-6492-0000-0000-000000000003', 8, 0, FROM_UNIXTIME(@ts - 3200), FROM_UNIXTIME(@ts - 3200)),
(UUID(), 'user-0001-0000-0000-000000000003', '《迷雾森林》的治愈感很强，艾琳的成长线写得很细腻', 'story', 'story-6492-0000-0000-000000000002', 15, 2, FROM_UNIXTIME(@ts - 2800), FROM_UNIXTIME(@ts - 2800)),
(UUID(), 'user-0001-0000-0000-000000000004', '白鹿的意象用得太好了，有种童话般的美感', 'storyboard', 'board-6492-0000-0000-000000000010', 6, 0, FROM_UNIXTIME(@ts - 2400), FROM_UNIXTIME(@ts - 2400)),
(UUID(), 'user-0001-0000-0000-000000000005', '《午夜电台》的悬疑氛围拉满！夜声到底是谁？', 'story', 'story-6492-0000-0000-000000000003', 18, 1, FROM_UNIXTIME(@ts - 2000), FROM_UNIXTIME(@ts - 2000)),
(UUID(), 'user-0001-0000-0000-000000000001', '镜子里没有倒影这个设定绝了，细思极恐', 'storyboard', 'board-6492-0000-0000-000000000019', 22, 0, FROM_UNIXTIME(@ts - 1600), FROM_UNIXTIME(@ts - 1600));

-- 12. King 的 100 条故事碎片
INSERT INTO fragments (id, creator_id, content, image_urls, visibility, source_type, is_converted, likes, comments, shares, views, created_at, updated_at) VALUES
('frag-6492-0000-0000-000000000001', @king_id, '【灵感】第1条记录。一个能看见死亡倒计时的人，发现自己的数字是问号。他试过一切办法，却无法得知自己的死期。直到他遇见了一个数字为零的人——那个人，是死神。', '["https://picsum.photos/seed/king-frag2/400/300"]', 'public', 'original', 0, 11, 3, 1, 123, 1772880180, 1772880180),
('frag-6492-0000-0000-000000000002', @king_id, '【场景】第2条记录。雨打在玻璃上，咖啡馆里只有她一个人。服务员递来一杯热可可：老样子？她点头，看向窗外。三年了，他再也没出现过。', '["https://picsum.photos/seed/king-frag3/400/300"]', 'public', 'original', 0, 12, 4, 2, 146, 1772876580, 1772876580),
('frag-6492-0000-0000-000000000003', @king_id, '【对话】第3条记录。你相信平行世界吗？相信。那你说，在另一个世界，我们会不会在一起？可能吧。那这个世界呢？', '["https://picsum.photos/seed/king-frag4/400/300"]', 'public', 'original', 0, 13, 5, 3, 169, 1772872980, 1772872980),
('frag-6492-0000-0000-000000000004', @king_id, '【设定】第4条记录。在这个世界，每个人出生时都会收到一封信，上面写着你会爱上的人的名字。但有些人收到的，是空白的。', '["https://picsum.photos/seed/king-frag5/400/300"]', 'public', 'original', 0, 14, 6, 4, 192, 1772869380, 1772869380),
('frag-6492-0000-0000-000000000005', @king_id, '【开篇】第5条记录。她醒来时，发现自己在别人的身体里。镜子里是一张陌生的脸，手机里是陌生的联系人。只有一件事是确定的：她必须在天亮前找到回去的方法。', '["https://picsum.photos/seed/king-frag6/400/300"]', 'public', 'original', 0, 15, 7, 0, 215, 1772865780, 1772865780),
('frag-6492-0000-0000-000000000006', @king_id, '【角色】第6条记录。一个不会说谎的骗子。他说的每句话都是真的，但人们从不相信。直到有一天，他决定说一个没人会信的真相。', '["https://picsum.photos/seed/king-frag7/400/300"]', 'public', 'original', 0, 16, 8, 1, 238, 1772862180, 1772862180),
('frag-6492-0000-0000-000000000007', @king_id, '【世界观】第7条记录。在这个城市，梦可以买卖。穷人们出售美梦换取温饱，富人们购买噩梦来体验刺激。而她，是一个造梦师。', '["https://picsum.photos/seed/king-frag8/400/300"]', 'public', 'original', 0, 17, 9, 2, 261, 1772858580, 1772858580),
('frag-6492-0000-0000-000000000008', @king_id, '【反转】第8条记录。侦探追查了十年的连环杀手，最后发现凶手是十年前的自己。时间旅行是存在的，但他不记得自己做过什么。', '["https://picsum.photos/seed/king-frag9/400/300"]', 'public', 'original', 0, 18, 10, 3, 284, 1772854980, 1772854980),
('frag-6492-0000-0000-000000000009', @king_id, '【氛围】第9条记录。午夜的地铁站，最后一班车已经开走。她坐在长椅上，听着自己的脚步声在空荡的站台回荡。', '["https://picsum.photos/seed/king-frag10/400/300"]', 'public', 'original', 0, 19, 11, 4, 307, 1772851380, 1772851380),
('frag-6492-0000-0000-000000000010', @king_id, '【悬念】第10条记录。他收到一个包裹，里面是一本日记。每一页都写着明天的日期，和明天会发生的事。最后一页是空白的，日期是今天。', '["https://picsum.photos/seed/king-frag11/400/300"]', 'public', 'original', 0, 20, 12, 0, 330, 1772847780, 1772847780),
('frag-6492-0000-0000-000000000011', @king_id, '【灵感】第11条记录。一个能看见死亡倒计时的人，发现自己的数字是问号。他试过一切办法，却无法得知自己的死期。直到他遇见了一个数字为零的人——那个人，是死神。', '["https://picsum.photos/seed/king-frag12/400/300"]', 'public', 'original', 0, 21, 13, 1, 353, 1772844180, 1772844180),
('frag-6492-0000-0000-000000000012', @king_id, '【场景】第12条记录。雨打在玻璃上，咖啡馆里只有她一个人。服务员递来一杯热可可：老样子？她点头，看向窗外。三年了，他再也没出现过。', '["https://picsum.photos/seed/king-frag13/400/300"]', 'public', 'original', 0, 22, 14, 2, 376, 1772840580, 1772840580),
('frag-6492-0000-0000-000000000013', @king_id, '【对话】第13条记录。你相信平行世界吗？相信。那你说，在另一个世界，我们会不会在一起？可能吧。那这个世界呢？', '["https://picsum.photos/seed/king-frag14/400/300"]', 'public', 'original', 0, 23, 15, 3, 399, 1772836980, 1772836980),
('frag-6492-0000-0000-000000000014', @king_id, '【设定】第14条记录。在这个世界，每个人出生时都会收到一封信，上面写着你会爱上的人的名字。但有些人收到的，是空白的。', '["https://picsum.photos/seed/king-frag15/400/300"]', 'public', 'original', 0, 24, 16, 4, 422, 1772833380, 1772833380),
('frag-6492-0000-0000-000000000015', @king_id, '【开篇】第15条记录。她醒来时，发现自己在别人的身体里。镜子里是一张陌生的脸，手机里是陌生的联系人。只有一件事是确定的：她必须在天亮前找到回去的方法。', '["https://picsum.photos/seed/king-frag16/400/300"]', 'public', 'original', 0, 25, 2, 0, 445, 1772829780, 1772829780),
('frag-6492-0000-0000-000000000016', @king_id, '【角色】第16条记录。一个不会说谎的骗子。他说的每句话都是真的，但人们从不相信。直到有一天，他决定说一个没人会信的真相。', '["https://picsum.photos/seed/king-frag17/400/300"]', 'public', 'original', 0, 26, 3, 1, 468, 1772826180, 1772826180),
('frag-6492-0000-0000-000000000017', @king_id, '【世界观】第17条记录。在这个城市，梦可以买卖。穷人们出售美梦换取温饱，富人们购买噩梦来体验刺激。而她，是一个造梦师。', '["https://picsum.photos/seed/king-frag18/400/300"]', 'public', 'original', 0, 27, 4, 2, 491, 1772822580, 1772822580),
('frag-6492-0000-0000-000000000018', @king_id, '【反转】第18条记录。侦探追查了十年的连环杀手，最后发现凶手是十年前的自己。时间旅行是存在的，但他不记得自己做过什么。', '["https://picsum.photos/seed/king-frag19/400/300"]', 'public', 'original', 0, 28, 5, 3, 514, 1772818980, 1772818980),
('frag-6492-0000-0000-000000000019', @king_id, '【氛围】第19条记录。午夜的地铁站，最后一班车已经开走。她坐在长椅上，听着自己的脚步声在空荡的站台回荡。', '["https://picsum.photos/seed/king-frag20/400/300"]', 'public', 'original', 0, 29, 6, 4, 537, 1772815380, 1772815380),
('frag-6492-0000-0000-000000000020', @king_id, '【悬念】第20条记录。他收到一个包裹，里面是一本日记。每一页都写着明天的日期，和明天会发生的事。最后一页是空白的，日期是今天。', '["https://picsum.photos/seed/king-frag21/400/300"]', 'public', 'original', 0, 30, 7, 0, 560, 1772811780, 1772811780),
('frag-6492-0000-0000-000000000021', @king_id, '【灵感】第21条记录。一个能看见死亡倒计时的人，发现自己的数字是问号。他试过一切办法，却无法得知自己的死期。直到他遇见了一个数字为零的人——那个人，是死神。', '["https://picsum.photos/seed/king-frag22/400/300"]', 'public', 'original', 0, 31, 8, 1, 583, 1772808180, 1772808180),
('frag-6492-0000-0000-000000000022', @king_id, '【场景】第22条记录。雨打在玻璃上，咖啡馆里只有她一个人。服务员递来一杯热可可：老样子？她点头，看向窗外。三年了，他再也没出现过。', '["https://picsum.photos/seed/king-frag23/400/300"]', 'public', 'original', 0, 32, 9, 2, 106, 1772804580, 1772804580),
('frag-6492-0000-0000-000000000023', @king_id, '【对话】第23条记录。你相信平行世界吗？相信。那你说，在另一个世界，我们会不会在一起？可能吧。那这个世界呢？', '["https://picsum.photos/seed/king-frag24/400/300"]', 'public', 'original', 0, 33, 10, 3, 129, 1772800980, 1772800980),
('frag-6492-0000-0000-000000000024', @king_id, '【设定】第24条记录。在这个世界，每个人出生时都会收到一封信，上面写着你会爱上的人的名字。但有些人收到的，是空白的。', '["https://picsum.photos/seed/king-frag25/400/300"]', 'public', 'original', 0, 34, 11, 4, 152, 1772797380, 1772797380),
('frag-6492-0000-0000-000000000025', @king_id, '【开篇】第25条记录。她醒来时，发现自己在别人的身体里。镜子里是一张陌生的脸，手机里是陌生的联系人。只有一件事是确定的：她必须在天亮前找到回去的方法。', '["https://picsum.photos/seed/king-frag26/400/300"]', 'public', 'original', 0, 35, 12, 0, 175, 1772793780, 1772793780),
('frag-6492-0000-0000-000000000026', @king_id, '【角色】第26条记录。一个不会说谎的骗子。他说的每句话都是真的，但人们从不相信。直到有一天，他决定说一个没人会信的真相。', '["https://picsum.photos/seed/king-frag27/400/300"]', 'public', 'original', 0, 36, 13, 1, 198, 1772790180, 1772790180),
('frag-6492-0000-0000-000000000027', @king_id, '【世界观】第27条记录。在这个城市，梦可以买卖。穷人们出售美梦换取温饱，富人们购买噩梦来体验刺激。而她，是一个造梦师。', '["https://picsum.photos/seed/king-frag28/400/300"]', 'public', 'original', 0, 37, 14, 2, 221, 1772786580, 1772786580),
('frag-6492-0000-0000-000000000028', @king_id, '【反转】第28条记录。侦探追查了十年的连环杀手，最后发现凶手是十年前的自己。时间旅行是存在的，但他不记得自己做过什么。', '["https://picsum.photos/seed/king-frag29/400/300"]', 'public', 'original', 0, 38, 15, 3, 244, 1772782980, 1772782980),
('frag-6492-0000-0000-000000000029', @king_id, '【氛围】第29条记录。午夜的地铁站，最后一班车已经开走。她坐在长椅上，听着自己的脚步声在空荡的站台回荡。', '["https://picsum.photos/seed/king-frag30/400/300"]', 'public', 'original', 0, 39, 16, 4, 267, 1772779380, 1772779380),
('frag-6492-0000-0000-000000000030', @king_id, '【悬念】第30条记录。他收到一个包裹，里面是一本日记。每一页都写着明天的日期，和明天会发生的事。最后一页是空白的，日期是今天。', '["https://picsum.photos/seed/king-frag31/400/300"]', 'public', 'original', 0, 40, 2, 0, 290, 1772775780, 1772775780),
('frag-6492-0000-0000-000000000031', @king_id, '【灵感】第31条记录。一个能看见死亡倒计时的人，发现自己的数字是问号。他试过一切办法，却无法得知自己的死期。直到他遇见了一个数字为零的人——那个人，是死神。', '["https://picsum.photos/seed/king-frag32/400/300"]', 'public', 'original', 0, 41, 3, 1, 313, 1772772180, 1772772180),
('frag-6492-0000-0000-000000000032', @king_id, '【场景】第32条记录。雨打在玻璃上，咖啡馆里只有她一个人。服务员递来一杯热可可：老样子？她点头，看向窗外。三年了，他再也没出现过。', '["https://picsum.photos/seed/king-frag33/400/300"]', 'public', 'original', 0, 42, 4, 2, 336, 1772768580, 1772768580),
('frag-6492-0000-0000-000000000033', @king_id, '【对话】第33条记录。你相信平行世界吗？相信。那你说，在另一个世界，我们会不会在一起？可能吧。那这个世界呢？', '["https://picsum.photos/seed/king-frag34/400/300"]', 'public', 'original', 0, 43, 5, 3, 359, 1772764980, 1772764980),
('frag-6492-0000-0000-000000000034', @king_id, '【设定】第34条记录。在这个世界，每个人出生时都会收到一封信，上面写着你会爱上的人的名字。但有些人收到的，是空白的。', '["https://picsum.photos/seed/king-frag35/400/300"]', 'public', 'original', 0, 44, 6, 4, 382, 1772761380, 1772761380),
('frag-6492-0000-0000-000000000035', @king_id, '【开篇】第35条记录。她醒来时，发现自己在别人的身体里。镜子里是一张陌生的脸，手机里是陌生的联系人。只有一件事是确定的：她必须在天亮前找到回去的方法。', '["https://picsum.photos/seed/king-frag36/400/300"]', 'public', 'original', 0, 45, 7, 0, 405, 1772757780, 1772757780),
('frag-6492-0000-0000-000000000036', @king_id, '【角色】第36条记录。一个不会说谎的骗子。他说的每句话都是真的，但人们从不相信。直到有一天，他决定说一个没人会信的真相。', '["https://picsum.photos/seed/king-frag37/400/300"]', 'public', 'original', 0, 46, 8, 1, 428, 1772754180, 1772754180),
('frag-6492-0000-0000-000000000037', @king_id, '【世界观】第37条记录。在这个城市，梦可以买卖。穷人们出售美梦换取温饱，富人们购买噩梦来体验刺激。而她，是一个造梦师。', '["https://picsum.photos/seed/king-frag38/400/300"]', 'public', 'original', 0, 47, 9, 2, 451, 1772750580, 1772750580),
('frag-6492-0000-0000-000000000038', @king_id, '【反转】第38条记录。侦探追查了十年的连环杀手，最后发现凶手是十年前的自己。时间旅行是存在的，但他不记得自己做过什么。', '["https://picsum.photos/seed/king-frag39/400/300"]', 'public', 'original', 0, 48, 10, 3, 474, 1772746980, 1772746980),
('frag-6492-0000-0000-000000000039', @king_id, '【氛围】第39条记录。午夜的地铁站，最后一班车已经开走。她坐在长椅上，听着自己的脚步声在空荡的站台回荡。', '["https://picsum.photos/seed/king-frag40/400/300"]', 'public', 'original', 0, 49, 11, 4, 497, 1772743380, 1772743380),
('frag-6492-0000-0000-000000000040', @king_id, '【悬念】第40条记录。他收到一个包裹，里面是一本日记。每一页都写着明天的日期，和明天会发生的事。最后一页是空白的，日期是今天。', '["https://picsum.photos/seed/king-frag41/400/300"]', 'public', 'original', 0, 50, 12, 0, 520, 1772739780, 1772739780),
('frag-6492-0000-0000-000000000041', @king_id, '【灵感】第41条记录。一个能看见死亡倒计时的人，发现自己的数字是问号。他试过一切办法，却无法得知自己的死期。直到他遇见了一个数字为零的人——那个人，是死神。', '["https://picsum.photos/seed/king-frag42/400/300"]', 'public', 'original', 0, 51, 13, 1, 543, 1772736180, 1772736180),
('frag-6492-0000-0000-000000000042', @king_id, '【场景】第42条记录。雨打在玻璃上，咖啡馆里只有她一个人。服务员递来一杯热可可：老样子？她点头，看向窗外。三年了，他再也没出现过。', '["https://picsum.photos/seed/king-frag43/400/300"]', 'public', 'original', 0, 52, 14, 2, 566, 1772732580, 1772732580),
('frag-6492-0000-0000-000000000043', @king_id, '【对话】第43条记录。你相信平行世界吗？相信。那你说，在另一个世界，我们会不会在一起？可能吧。那这个世界呢？', '["https://picsum.photos/seed/king-frag44/400/300"]', 'public', 'original', 0, 53, 15, 3, 589, 1772728980, 1772728980),
('frag-6492-0000-0000-000000000044', @king_id, '【设定】第44条记录。在这个世界，每个人出生时都会收到一封信，上面写着你会爱上的人的名字。但有些人收到的，是空白的。', '["https://picsum.photos/seed/king-frag45/400/300"]', 'public', 'original', 0, 54, 16, 4, 112, 1772725380, 1772725380),
('frag-6492-0000-0000-000000000045', @king_id, '【开篇】第45条记录。她醒来时，发现自己在别人的身体里。镜子里是一张陌生的脸，手机里是陌生的联系人。只有一件事是确定的：她必须在天亮前找到回去的方法。', '["https://picsum.photos/seed/king-frag46/400/300"]', 'public', 'original', 0, 55, 2, 0, 135, 1772721780, 1772721780),
('frag-6492-0000-0000-000000000046', @king_id, '【角色】第46条记录。一个不会说谎的骗子。他说的每句话都是真的，但人们从不相信。直到有一天，他决定说一个没人会信的真相。', '["https://picsum.photos/seed/king-frag47/400/300"]', 'public', 'original', 0, 56, 3, 1, 158, 1772718180, 1772718180),
('frag-6492-0000-0000-000000000047', @king_id, '【世界观】第47条记录。在这个城市，梦可以买卖。穷人们出售美梦换取温饱，富人们购买噩梦来体验刺激。而她，是一个造梦师。', '["https://picsum.photos/seed/king-frag48/400/300"]', 'public', 'original', 0, 57, 4, 2, 181, 1772714580, 1772714580),
('frag-6492-0000-0000-000000000048', @king_id, '【反转】第48条记录。侦探追查了十年的连环杀手，最后发现凶手是十年前的自己。时间旅行是存在的，但他不记得自己做过什么。', '["https://picsum.photos/seed/king-frag49/400/300"]', 'public', 'original', 0, 58, 5, 3, 204, 1772710980, 1772710980),
('frag-6492-0000-0000-000000000049', @king_id, '【氛围】第49条记录。午夜的地铁站，最后一班车已经开走。她坐在长椅上，听着自己的脚步声在空荡的站台回荡。', '["https://picsum.photos/seed/king-frag50/400/300"]', 'public', 'original', 0, 59, 6, 4, 227, 1772707380, 1772707380),
('frag-6492-0000-0000-000000000050', @king_id, '【悬念】第50条记录。他收到一个包裹，里面是一本日记。每一页都写着明天的日期，和明天会发生的事。最后一页是空白的，日期是今天。', '["https://picsum.photos/seed/king-frag1/400/300"]', 'public', 'original', 0, 10, 7, 0, 250, 1772703780, 1772703780),
('frag-6492-0000-0000-000000000051', @king_id, '【灵感】第51条记录。一个能看见死亡倒计时的人，发现自己的数字是问号。他试过一切办法，却无法得知自己的死期。直到他遇见了一个数字为零的人——那个人，是死神。', '["https://picsum.photos/seed/king-frag2/400/300"]', 'public', 'original', 0, 11, 8, 1, 273, 1772700180, 1772700180),
('frag-6492-0000-0000-000000000052', @king_id, '【场景】第52条记录。雨打在玻璃上，咖啡馆里只有她一个人。服务员递来一杯热可可：老样子？她点头，看向窗外。三年了，他再也没出现过。', '["https://picsum.photos/seed/king-frag3/400/300"]', 'public', 'original', 0, 12, 9, 2, 296, 1772696580, 1772696580),
('frag-6492-0000-0000-000000000053', @king_id, '【对话】第53条记录。你相信平行世界吗？相信。那你说，在另一个世界，我们会不会在一起？可能吧。那这个世界呢？', '["https://picsum.photos/seed/king-frag4/400/300"]', 'public', 'original', 0, 13, 10, 3, 319, 1772692980, 1772692980),
('frag-6492-0000-0000-000000000054', @king_id, '【设定】第54条记录。在这个世界，每个人出生时都会收到一封信，上面写着你会爱上的人的名字。但有些人收到的，是空白的。', '["https://picsum.photos/seed/king-frag5/400/300"]', 'public', 'original', 0, 14, 11, 4, 342, 1772689380, 1772689380),
('frag-6492-0000-0000-000000000055', @king_id, '【开篇】第55条记录。她醒来时，发现自己在别人的身体里。镜子里是一张陌生的脸，手机里是陌生的联系人。只有一件事是确定的：她必须在天亮前找到回去的方法。', '["https://picsum.photos/seed/king-frag6/400/300"]', 'public', 'original', 0, 15, 12, 0, 365, 1772685780, 1772685780),
('frag-6492-0000-0000-000000000056', @king_id, '【角色】第56条记录。一个不会说谎的骗子。他说的每句话都是真的，但人们从不相信。直到有一天，他决定说一个没人会信的真相。', '["https://picsum.photos/seed/king-frag7/400/300"]', 'public', 'original', 0, 16, 13, 1, 388, 1772682180, 1772682180),
('frag-6492-0000-0000-000000000057', @king_id, '【世界观】第57条记录。在这个城市，梦可以买卖。穷人们出售美梦换取温饱，富人们购买噩梦来体验刺激。而她，是一个造梦师。', '["https://picsum.photos/seed/king-frag8/400/300"]', 'public', 'original', 0, 17, 14, 2, 411, 1772678580, 1772678580),
('frag-6492-0000-0000-000000000058', @king_id, '【反转】第58条记录。侦探追查了十年的连环杀手，最后发现凶手是十年前的自己。时间旅行是存在的，但他不记得自己做过什么。', '["https://picsum.photos/seed/king-frag9/400/300"]', 'public', 'original', 0, 18, 15, 3, 434, 1772674980, 1772674980),
('frag-6492-0000-0000-000000000059', @king_id, '【氛围】第59条记录。午夜的地铁站，最后一班车已经开走。她坐在长椅上，听着自己的脚步声在空荡的站台回荡。', '["https://picsum.photos/seed/king-frag10/400/300"]', 'public', 'original', 0, 19, 16, 4, 457, 1772671380, 1772671380),
('frag-6492-0000-0000-000000000060', @king_id, '【悬念】第60条记录。他收到一个包裹，里面是一本日记。每一页都写着明天的日期，和明天会发生的事。最后一页是空白的，日期是今天。', '["https://picsum.photos/seed/king-frag11/400/300"]', 'public', 'original', 0, 20, 2, 0, 480, 1772667780, 1772667780),
('frag-6492-0000-0000-000000000061', @king_id, '【灵感】第61条记录。一个能看见死亡倒计时的人，发现自己的数字是问号。他试过一切办法，却无法得知自己的死期。直到他遇见了一个数字为零的人——那个人，是死神。', '["https://picsum.photos/seed/king-frag12/400/300"]', 'public', 'original', 0, 21, 3, 1, 503, 1772664180, 1772664180),
('frag-6492-0000-0000-000000000062', @king_id, '【场景】第62条记录。雨打在玻璃上，咖啡馆里只有她一个人。服务员递来一杯热可可：老样子？她点头，看向窗外。三年了，他再也没出现过。', '["https://picsum.photos/seed/king-frag13/400/300"]', 'public', 'original', 0, 22, 4, 2, 526, 1772660580, 1772660580),
('frag-6492-0000-0000-000000000063', @king_id, '【对话】第63条记录。你相信平行世界吗？相信。那你说，在另一个世界，我们会不会在一起？可能吧。那这个世界呢？', '["https://picsum.photos/seed/king-frag14/400/300"]', 'public', 'original', 0, 23, 5, 3, 549, 1772656980, 1772656980),
('frag-6492-0000-0000-000000000064', @king_id, '【设定】第64条记录。在这个世界，每个人出生时都会收到一封信，上面写着你会爱上的人的名字。但有些人收到的，是空白的。', '["https://picsum.photos/seed/king-frag15/400/300"]', 'public', 'original', 0, 24, 6, 4, 572, 1772653380, 1772653380),
('frag-6492-0000-0000-000000000065', @king_id, '【开篇】第65条记录。她醒来时，发现自己在别人的身体里。镜子里是一张陌生的脸，手机里是陌生的联系人。只有一件事是确定的：她必须在天亮前找到回去的方法。', '["https://picsum.photos/seed/king-frag16/400/300"]', 'public', 'original', 0, 25, 7, 0, 595, 1772649780, 1772649780),
('frag-6492-0000-0000-000000000066', @king_id, '【角色】第66条记录。一个不会说谎的骗子。他说的每句话都是真的，但人们从不相信。直到有一天，他决定说一个没人会信的真相。', '["https://picsum.photos/seed/king-frag17/400/300"]', 'public', 'original', 0, 26, 8, 1, 118, 1772646180, 1772646180),
('frag-6492-0000-0000-000000000067', @king_id, '【世界观】第67条记录。在这个城市，梦可以买卖。穷人们出售美梦换取温饱，富人们购买噩梦来体验刺激。而她，是一个造梦师。', '["https://picsum.photos/seed/king-frag18/400/300"]', 'public', 'original', 0, 27, 9, 2, 141, 1772642580, 1772642580),
('frag-6492-0000-0000-000000000068', @king_id, '【反转】第68条记录。侦探追查了十年的连环杀手，最后发现凶手是十年前的自己。时间旅行是存在的，但他不记得自己做过什么。', '["https://picsum.photos/seed/king-frag19/400/300"]', 'public', 'original', 0, 28, 10, 3, 164, 1772638980, 1772638980),
('frag-6492-0000-0000-000000000069', @king_id, '【氛围】第69条记录。午夜的地铁站，最后一班车已经开走。她坐在长椅上，听着自己的脚步声在空荡的站台回荡。', '["https://picsum.photos/seed/king-frag20/400/300"]', 'public', 'original', 0, 29, 11, 4, 187, 1772635380, 1772635380),
('frag-6492-0000-0000-000000000070', @king_id, '【悬念】第70条记录。他收到一个包裹，里面是一本日记。每一页都写着明天的日期，和明天会发生的事。最后一页是空白的，日期是今天。', '["https://picsum.photos/seed/king-frag21/400/300"]', 'public', 'original', 0, 30, 12, 0, 210, 1772631780, 1772631780),
('frag-6492-0000-0000-000000000071', @king_id, '【灵感】第71条记录。一个能看见死亡倒计时的人，发现自己的数字是问号。他试过一切办法，却无法得知自己的死期。直到他遇见了一个数字为零的人——那个人，是死神。', '["https://picsum.photos/seed/king-frag22/400/300"]', 'public', 'original', 0, 31, 13, 1, 233, 1772628180, 1772628180),
('frag-6492-0000-0000-000000000072', @king_id, '【场景】第72条记录。雨打在玻璃上，咖啡馆里只有她一个人。服务员递来一杯热可可：老样子？她点头，看向窗外。三年了，他再也没出现过。', '["https://picsum.photos/seed/king-frag23/400/300"]', 'public', 'original', 0, 32, 14, 2, 256, 1772624580, 1772624580),
('frag-6492-0000-0000-000000000073', @king_id, '【对话】第73条记录。你相信平行世界吗？相信。那你说，在另一个世界，我们会不会在一起？可能吧。那这个世界呢？', '["https://picsum.photos/seed/king-frag24/400/300"]', 'public', 'original', 0, 33, 15, 3, 279, 1772620980, 1772620980),
('frag-6492-0000-0000-000000000074', @king_id, '【设定】第74条记录。在这个世界，每个人出生时都会收到一封信，上面写着你会爱上的人的名字。但有些人收到的，是空白的。', '["https://picsum.photos/seed/king-frag25/400/300"]', 'public', 'original', 0, 34, 16, 4, 302, 1772617380, 1772617380),
('frag-6492-0000-0000-000000000075', @king_id, '【开篇】第75条记录。她醒来时，发现自己在别人的身体里。镜子里是一张陌生的脸，手机里是陌生的联系人。只有一件事是确定的：她必须在天亮前找到回去的方法。', '["https://picsum.photos/seed/king-frag26/400/300"]', 'public', 'original', 0, 35, 2, 0, 325, 1772613780, 1772613780),
('frag-6492-0000-0000-000000000076', @king_id, '【角色】第76条记录。一个不会说谎的骗子。他说的每句话都是真的，但人们从不相信。直到有一天，他决定说一个没人会信的真相。', '["https://picsum.photos/seed/king-frag27/400/300"]', 'public', 'original', 0, 36, 3, 1, 348, 1772610180, 1772610180),
('frag-6492-0000-0000-000000000077', @king_id, '【世界观】第77条记录。在这个城市，梦可以买卖。穷人们出售美梦换取温饱，富人们购买噩梦来体验刺激。而她，是一个造梦师。', '["https://picsum.photos/seed/king-frag28/400/300"]', 'public', 'original', 0, 37, 4, 2, 371, 1772606580, 1772606580),
('frag-6492-0000-0000-000000000078', @king_id, '【反转】第78条记录。侦探追查了十年的连环杀手，最后发现凶手是十年前的自己。时间旅行是存在的，但他不记得自己做过什么。', '["https://picsum.photos/seed/king-frag29/400/300"]', 'public', 'original', 0, 38, 5, 3, 394, 1772602980, 1772602980),
('frag-6492-0000-0000-000000000079', @king_id, '【氛围】第79条记录。午夜的地铁站，最后一班车已经开走。她坐在长椅上，听着自己的脚步声在空荡的站台回荡。', '["https://picsum.photos/seed/king-frag30/400/300"]', 'public', 'original', 0, 39, 6, 4, 417, 1772599380, 1772599380),
('frag-6492-0000-0000-000000000080', @king_id, '【悬念】第80条记录。他收到一个包裹，里面是一本日记。每一页都写着明天的日期，和明天会发生的事。最后一页是空白的，日期是今天。', '["https://picsum.photos/seed/king-frag31/400/300"]', 'public', 'original', 0, 40, 7, 0, 440, 1772595780, 1772595780),
('frag-6492-0000-0000-000000000081', @king_id, '【灵感】第81条记录。一个能看见死亡倒计时的人，发现自己的数字是问号。他试过一切办法，却无法得知自己的死期。直到他遇见了一个数字为零的人——那个人，是死神。', '["https://picsum.photos/seed/king-frag32/400/300"]', 'public', 'original', 0, 41, 8, 1, 463, 1772592180, 1772592180),
('frag-6492-0000-0000-000000000082', @king_id, '【场景】第82条记录。雨打在玻璃上，咖啡馆里只有她一个人。服务员递来一杯热可可：老样子？她点头，看向窗外。三年了，他再也没出现过。', '["https://picsum.photos/seed/king-frag33/400/300"]', 'public', 'original', 0, 42, 9, 2, 486, 1772588580, 1772588580),
('frag-6492-0000-0000-000000000083', @king_id, '【对话】第83条记录。你相信平行世界吗？相信。那你说，在另一个世界，我们会不会在一起？可能吧。那这个世界呢？', '["https://picsum.photos/seed/king-frag34/400/300"]', 'public', 'original', 0, 43, 10, 3, 509, 1772584980, 1772584980),
('frag-6492-0000-0000-000000000084', @king_id, '【设定】第84条记录。在这个世界，每个人出生时都会收到一封信，上面写着你会爱上的人的名字。但有些人收到的，是空白的。', '["https://picsum.photos/seed/king-frag35/400/300"]', 'public', 'original', 0, 44, 11, 4, 532, 1772581380, 1772581380),
('frag-6492-0000-0000-000000000085', @king_id, '【开篇】第85条记录。她醒来时，发现自己在别人的身体里。镜子里是一张陌生的脸，手机里是陌生的联系人。只有一件事是确定的：她必须在天亮前找到回去的方法。', '["https://picsum.photos/seed/king-frag36/400/300"]', 'public', 'original', 0, 45, 12, 0, 555, 1772577780, 1772577780),
('frag-6492-0000-0000-000000000086', @king_id, '【角色】第86条记录。一个不会说谎的骗子。他说的每句话都是真的，但人们从不相信。直到有一天，他决定说一个没人会信的真相。', '["https://picsum.photos/seed/king-frag37/400/300"]', 'public', 'original', 0, 46, 13, 1, 578, 1772574180, 1772574180),
('frag-6492-0000-0000-000000000087', @king_id, '【世界观】第87条记录。在这个城市，梦可以买卖。穷人们出售美梦换取温饱，富人们购买噩梦来体验刺激。而她，是一个造梦师。', '["https://picsum.photos/seed/king-frag38/400/300"]', 'public', 'original', 0, 47, 14, 2, 101, 1772570580, 1772570580),
('frag-6492-0000-0000-000000000088', @king_id, '【反转】第88条记录。侦探追查了十年的连环杀手，最后发现凶手是十年前的自己。时间旅行是存在的，但他不记得自己做过什么。', '["https://picsum.photos/seed/king-frag39/400/300"]', 'public', 'original', 0, 48, 15, 3, 124, 1772566980, 1772566980),
('frag-6492-0000-0000-000000000089', @king_id, '【氛围】第89条记录。午夜的地铁站，最后一班车已经开走。她坐在长椅上，听着自己的脚步声在空荡的站台回荡。', '["https://picsum.photos/seed/king-frag40/400/300"]', 'public', 'original', 0, 49, 16, 4, 147, 1772563380, 1772563380),
('frag-6492-0000-0000-000000000090', @king_id, '【悬念】第90条记录。他收到一个包裹，里面是一本日记。每一页都写着明天的日期，和明天会发生的事。最后一页是空白的，日期是今天。', '["https://picsum.photos/seed/king-frag41/400/300"]', 'public', 'original', 0, 50, 2, 0, 170, 1772559780, 1772559780),
('frag-6492-0000-0000-000000000091', @king_id, '【灵感】第91条记录。一个能看见死亡倒计时的人，发现自己的数字是问号。他试过一切办法，却无法得知自己的死期。直到他遇见了一个数字为零的人——那个人，是死神。', '["https://picsum.photos/seed/king-frag42/400/300"]', 'public', 'original', 0, 51, 3, 1, 193, 1772556180, 1772556180),
('frag-6492-0000-0000-000000000092', @king_id, '【场景】第92条记录。雨打在玻璃上，咖啡馆里只有她一个人。服务员递来一杯热可可：老样子？她点头，看向窗外。三年了，他再也没出现过。', '["https://picsum.photos/seed/king-frag43/400/300"]', 'public', 'original', 0, 52, 4, 2, 216, 1772552580, 1772552580),
('frag-6492-0000-0000-000000000093', @king_id, '【对话】第93条记录。你相信平行世界吗？相信。那你说，在另一个世界，我们会不会在一起？可能吧。那这个世界呢？', '["https://picsum.photos/seed/king-frag44/400/300"]', 'public', 'original', 0, 53, 5, 3, 239, 1772548980, 1772548980),
('frag-6492-0000-0000-000000000094', @king_id, '【设定】第94条记录。在这个世界，每个人出生时都会收到一封信，上面写着你会爱上的人的名字。但有些人收到的，是空白的。', '["https://picsum.photos/seed/king-frag45/400/300"]', 'public', 'original', 0, 54, 6, 4, 262, 1772545380, 1772545380),
('frag-6492-0000-0000-000000000095', @king_id, '【开篇】第95条记录。她醒来时，发现自己在别人的身体里。镜子里是一张陌生的脸，手机里是陌生的联系人。只有一件事是确定的：她必须在天亮前找到回去的方法。', '["https://picsum.photos/seed/king-frag46/400/300"]', 'public', 'original', 0, 55, 7, 0, 285, 1772541780, 1772541780),
('frag-6492-0000-0000-000000000096', @king_id, '【角色】第96条记录。一个不会说谎的骗子。他说的每句话都是真的，但人们从不相信。直到有一天，他决定说一个没人会信的真相。', '["https://picsum.photos/seed/king-frag47/400/300"]', 'public', 'original', 0, 56, 8, 1, 308, 1772538180, 1772538180),
('frag-6492-0000-0000-000000000097', @king_id, '【世界观】第97条记录。在这个城市，梦可以买卖。穷人们出售美梦换取温饱，富人们购买噩梦来体验刺激。而她，是一个造梦师。', '["https://picsum.photos/seed/king-frag48/400/300"]', 'public', 'original', 0, 57, 9, 2, 331, 1772534580, 1772534580),
('frag-6492-0000-0000-000000000098', @king_id, '【反转】第98条记录。侦探追查了十年的连环杀手，最后发现凶手是十年前的自己。时间旅行是存在的，但他不记得自己做过什么。', '["https://picsum.photos/seed/king-frag49/400/300"]', 'public', 'original', 0, 58, 10, 3, 354, 1772530980, 1772530980),
('frag-6492-0000-0000-000000000099', @king_id, '【氛围】第99条记录。午夜的地铁站，最后一班车已经开走。她坐在长椅上，听着自己的脚步声在空荡的站台回荡。', '["https://picsum.photos/seed/king-frag50/400/300"]', 'public', 'original', 0, 59, 11, 4, 377, 1772527380, 1772527380),
('frag-6492-0000-0000-000000000100', @king_id, '【悬念】第100条记录。他收到一个包裹，里面是一本日记。每一页都写着明天的日期，和明天会发生的事。最后一页是空白的，日期是今天。', '["https://picsum.photos/seed/king-frag1/400/300"]', 'public', 'original', 0, 10, 12, 0, 400, 1772523780, 1772523780)
ON DUPLICATE KEY UPDATE updated_at = VALUES(updated_at);

-- 13. 故事碎片评论 (其他用户对 King 碎片的评论)
INSERT INTO fragment_comments (id, fragment_id, user_id, content, parent_id, created_at, updated_at) VALUES
(UUID(), 'frag-6492-0000-0000-000000000001', 'user-0001-0000-0000-000000000001', '死亡倒计时的设定太有意思了！期待看到完整故事', NULL, @ts - 3600, @ts - 3600),
(UUID(), 'frag-6492-0000-0000-000000000001', 'user-0001-0000-0000-000000000002', '问号和死神相遇的设定很有张力', NULL, @ts - 3200, @ts - 3200),
(UUID(), 'frag-6492-0000-0000-000000000001', 'user-0001-0000-0000-000000000003', '同感！这个灵感可以扩展成很多故事', NULL, @ts - 2800, @ts - 2800),
(UUID(), 'frag-6492-0000-0000-000000000002', 'user-0001-0000-0000-000000000004', '咖啡馆的三年等待，好戳', NULL, @ts - 3600, @ts - 3600),
(UUID(), 'frag-6492-0000-0000-000000000002', 'user-0001-0000-0000-000000000005', '老样子这个细节太催泪了', NULL, @ts - 3400, @ts - 3400),
(UUID(), 'frag-6492-0000-0000-000000000003', 'user-0001-0000-0000-000000000001', '平行世界的对话太有哲学感了', NULL, @ts - 3000, @ts - 3000),
(UUID(), 'frag-6492-0000-0000-000000000004', 'user-0001-0000-0000-000000000002', '空白信件的设定很浪漫', NULL, @ts - 2600, @ts - 2600),
(UUID(), 'frag-6492-0000-0000-000000000005', 'user-0001-0000-0000-000000000003', '身体互换的设定很新颖！', NULL, @ts - 2400, @ts - 2400),
(UUID(), 'frag-6492-0000-0000-000000000006', 'user-0001-0000-0000-000000000004', '不会说谎的骗子，这个矛盾很有张力', NULL, @ts - 2200, @ts - 2200),
(UUID(), 'frag-6492-0000-0000-000000000007', 'user-0001-0000-0000-000000000005', '造梦师的世界观太酷了', NULL, @ts - 2000, @ts - 2000),
(UUID(), 'frag-6492-0000-0000-000000000008', 'user-0001-0000-0000-000000000001', '时间旅行+凶手是自己的反转绝了', NULL, @ts - 1800, @ts - 1800),
(UUID(), 'frag-6492-0000-0000-000000000009', 'user-0001-0000-0000-000000000002', '午夜地铁站的氛围感拉满', NULL, @ts - 1600, @ts - 1600),
(UUID(), 'frag-6492-0000-0000-000000000010', 'user-0001-0000-0000-000000000003', '预知日记的悬念太强了', NULL, @ts - 1400, @ts - 1400),
(UUID(), 'frag-6492-0000-0000-000000000015', 'user-0001-0000-0000-000000000004', '开篇的设定很吸引人', NULL, @ts - 1200, @ts - 1200),
(UUID(), 'frag-6492-0000-0000-000000000020', 'user-0001-0000-0000-000000000005', '悬念感十足！', NULL, @ts - 1000, @ts - 1000),
(UUID(), 'frag-6492-0000-0000-000000000025', 'user-0001-0000-0000-000000000001', 'King 的创作风格很统一', NULL, @ts - 800, @ts - 800),
(UUID(), 'frag-6492-0000-0000-000000000030', 'user-0001-0000-0000-000000000002', '期待更多碎片！', NULL, @ts - 600, @ts - 600),
(UUID(), 'frag-6492-0000-0000-000000000050', 'user-0001-0000-0000-000000000003', '第50条了，创作力惊人', NULL, @ts - 400, @ts - 400),
(UUID(), 'frag-6492-0000-0000-000000000075', 'user-0001-0000-0000-000000000004', '持续关注中', NULL, @ts - 200, @ts - 200),
(UUID(), 'frag-6492-0000-0000-000000000100', 'user-0001-0000-0000-000000000005', '第100条！恭喜完结这一批创作', NULL, @ts, @ts);

-- ============================================================
-- King mock data inserted. 最终数据统计:
-- ============================================================
SELECT 'King mock data inserted successfully!' AS status;
SELECT
    (SELECT COUNT(*) FROM users) AS users,
    (SELECT COUNT(*) FROM stories) AS stories,
    (SELECT COUNT(*) FROM storyboards) AS storyboards,
    (SELECT COUNT(*) FROM characters) AS characters,
    (SELECT COUNT(*) FROM fragments) AS fragments,
    (SELECT COUNT(*) FROM comments) AS comments,
    (SELECT COUNT(*) FROM fragment_comments) AS fragment_comments;
