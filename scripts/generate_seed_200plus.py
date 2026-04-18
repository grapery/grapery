#!/usr/bin/env python3
"""Generate scripts/seed_dev_diverse_200plus.sql — run from repo grapery/: python3 scripts/generate_seed_200plus.py"""
from __future__ import annotations

import pathlib
import uuid

OUT = pathlib.Path(__file__).resolve().parent / "seed_dev_diverse_200plus.sql"

PW = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZRGdjGj/n3.QGhS5ThOy6CxbLWPPm"


def uid(kind: str, n: int) -> str:
    """Deterministic UUIDv5 for stable FKs across regenerations."""
    return str(uuid.uuid5(uuid.NAMESPACE_URL, f"seed200:{kind}:{n}"))


def main() -> None:
    nu = 20
    ns = 25
    nf = 35
    users = [uid("userseed", i) for i in range(1, nu + 1)]
    stories = [uid("storyseed", i) for i in range(1, ns + 1)]
    chars = [uid("charseed", i) for i in range(1, ns + 1)]
    boards = [uid("boardseed", i) for i in range(1, ns + 1)]
    scenes = [uid("scnseed", i) for i in range(1, ns + 1)]
    frags = [uid("fragseed", i) for i in range(1, nf + 1)]
    tags = [uid("tagseed", i) for i in range(1, 11)]

    lines: list[str] = [
        "-- seed_dev_diverse_200plus.sql",
        "-- Run AFTER migrations: mysql ... grapery < scripts/seed_dev_diverse_200plus.sql",
        "-- Idempotent: clears only seed-owned rows (IDs from uuid5 namespace seed200:...).",
        "SET NAMES utf8mb4;",
        "SET FOREIGN_KEY_CHECKS = 0;",
        "SET SQL_SAFE_UPDATES = 0;",
        "DELETE FROM notifications WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'seed200_u%');",
        "DELETE FROM search_histories WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'seed200_u%');",
        "DELETE FROM view_histories WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'seed200_u%');",
        "DELETE FROM story_follows WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'seed200_u%');",
        "DELETE FROM user_follows WHERE follower_id IN (SELECT id FROM users WHERE username LIKE 'seed200_u%') "
        "OR followee_id IN (SELECT id FROM users WHERE username LIKE 'seed200_u%');",
        "DELETE FROM fragment_comments WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'seed200_u%');",
        "DELETE FROM fragment_likes WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'seed200_u%');",
        "DELETE FROM bookmarks WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'seed200_u%');",
        "DELETE FROM likes WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'seed200_u%');",
        "DELETE FROM storyboard_likes WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'seed200_u%');",
        "DELETE FROM story_likes WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'seed200_u%');",
        "DELETE FROM story_tags WHERE story_id IN (SELECT id FROM stories WHERE title LIKE 'Seed Story %');",
        "DELETE FROM tags WHERE name LIKE 'seed-%';",
        "DELETE FROM fragments WHERE creator_id IN (SELECT id FROM users WHERE username LIKE 'seed200_u%');",
        "DELETE FROM storyboard_scenes WHERE storyboard_id IN (SELECT id FROM storyboards WHERE title LIKE 'Board %');",
        "DELETE FROM storyboards WHERE title LIKE 'Board %';",
        "DELETE FROM characters WHERE name LIKE 'Character %';",
        "DELETE FROM stories WHERE title LIKE 'Seed Story %';",
        "DELETE FROM user_settings WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'seed200_u%');",
        "DELETE FROM users WHERE username LIKE 'seed200_u%';",
        "SET SQL_SAFE_UPDATES = 1;",
        "",
    ]

    # 1) Users (20)
    uv = []
    for i, u in enumerate(users, start=1):
        uv.append(
            f"('{u}', 'seed200_u{i:02d}', 'seed200_u{i:02d}@example.com', '{PW}', 'Seed User {i}', "
            f"'https://picsum.photos/seed/s{i}/200/200', '', 'Bulk seed user {i}', '', '', "
            f"10, 5, 0, 0, 'active', 1, UNIX_TIMESTAMP() * 1000, {i * 100}, 'SEED{i:04d}', "
            f"UNIX_TIMESTAMP(), UNIX_TIMESTAMP())"
        )
    lines.append("INSERT INTO users (id, username, email, password_hash, display_name, avatar, background, bio, location, website, followers, following, storyboard_count, fragments_count, status, email_verified, last_login_at, points, referral_code, created_at, updated_at) VALUES")
    lines.append(",\n".join(uv) + ";")
    lines.append("")

    # 2) User settings (20)
    sv = []
    for u in users:
        sv.append(
            f"(UUID(), '{u}', 'zh-CN', 'system', 'medium', 0, 'public', 'public', 'public', "
            f"'everyone', 'everyone', 'followers_only', 1, 1, 1, 1, 1, 1, 1, "
            f"CAST('{{}}' AS JSON), CAST('[]' AS JSON), UNIX_TIMESTAMP(), NULL)"
        )
    lines.append(
        "INSERT INTO user_settings (id, user_id, language, theme, font_size, data_saver, profile_visibility, "
        "default_story_visibility, default_fragment_visibility, allow_follow_from, allow_comments_from, allow_messages_from, "
        "show_online_status, show_read_receipts, show_public_stories, show_public_fragments, show_public_bookmarks, "
        "ai_enabled, a_idata_sharing, notification_settings, preferred_genres_json, updated_at, deleted_at) VALUES"
    )
    lines.append(",\n".join(sv) + ";")
    lines.append("")

    # 3) Stories (25)
    stv = []
    for i, sid in enumerate(stories, start=1):
        aid = users[(i - 1) % nu]
        stv.append(
            f"('{sid}', 'Seed Story {i}', 'Description for seed story {i}', 'https://picsum.photos/seed/st{i}/400/600', "
            f"'{aid}', NULL, {i * 3}, {i}, {i % 5}, 0, {1 + i % 5}, 3, 'fantasy', NULL, 'published', 0, 'public', 1, 1, "
            f"NULL, NOW(3), NOW(3), NULL)"
        )
    lines.append(
        "INSERT INTO stories (id, title, description, cover_image, author_id, source_fragment_id, likes, followers, saves, "
        "panels, storyboard_count, default_scene_count, genre, style, status, is_collaboration_open, visibility, use_ai, "
        "ai_enabled, ai_assistance_options, created_at, updated_at, deleted_at) VALUES"
    )
    lines.append(",\n".join(stv) + ";")
    lines.append("")

    # 4) Characters (25) — one per story (column order matches DESCRIBE characters)
    cv = []
    for i, (cid, sid) in enumerate(zip(chars, stories), start=1):
        aid = users[(i - 1) % nu]
        cv.append(
            f"('{cid}', '{sid}', 'Character {i}', 'Bio for character {i}', 'https://picsum.photos/seed/c{i}/200/200', "
            f"NULL, NULL, 0, NULL, 'none', NULL, '{aid}', 'Bold', 'Unknown', NULL, NULL, NULL, NULL, NULL, NULL, NULL, "
            f"'protagonist', 'manual', NULL, NULL, '{aid}', '{aid}', 0, 0, 0, 0, 0, NULL, NULL, 1, 'creator_only', NULL, 0, "
            f"NOW(3), NOW(3), NULL, NULL, NULL, NULL, NULL, NULL)"
        )
    lines.append(
        "INSERT INTO characters (id, story_id, name, description, avatar, poster, portrait, needs_portrait, reference_image, "
        "portrait_generation_status, views_json, author_id, personality, background, short_term_goal, long_term_goal, "
        "handling_style, cognition_range, ability_features, appearance, dress_preference, role, source_type, source_prompt, "
        "source_image, created_by, last_edited_by, likes, comments, shares, followers, stories, traits, skills, is_public, "
        "poster_creation_permission, origin_story_id, is_cameo, created_at, updated_at, deleted_at, portrait_style, "
        "portrait_background, portrait_lighting, portrait_angle, portrait_expression) VALUES"
    )
    lines.append(",\n".join(cv) + ";")
    lines.append("")

    # 5) Storyboards (25)
    bv = []
    for i, bid in enumerate(boards, start=1):
        sid = stories[i - 1]
        cid = users[(i - 1) % nu]
        bv.append(
            f"('{bid}', '{sid}', NULL, '{cid}', 'Board {i}', 'Content for board {i}.', 'raw seed {i}', 0, 1, 3, "
            f"'published', 5, {i % 7}, {i % 3}, 0, {i * 10}, 0, 0, NULL, NULL, NOW(3), NOW(3), NULL)"
        )
    lines.append(
        "INSERT INTO storyboards (id, story_id, parent_id, creator_id, title, content, raw_input, is_standalone, "
        "is_ai_generated, scene_count, workflow_status, current_step, likes, comments, shares, fork_count, views, "
        "token_consumption, fate_snapshot, fate_snapshot_hash, created_at, updated_at, deleted_at) VALUES"
    )
    lines.append(",\n".join(bv) + ";")
    lines.append("")

    # 6) Storyboard scenes (25)
    scv = []
    for i, scid in enumerate(scenes, start=1):
        bid = boards[i - 1]
        scv.append(
            f"('{scid}', '{bid}', NULL, 1, 'Scene {i}', 'Beat {i}', 'https://picsum.photos/seed/sc{i}/800/450', '', "
            f"'loc{i}', 'day', CAST('[\"Hero\"]' AS JSON), 'calm', 1, 0, NULL, NULL, NULL, NOW(3), NOW(3), NULL, "
            f"NULL, NULL, NULL)"
        )
    lines.append(
        "INSERT INTO storyboard_scenes (id, storyboard_id, story_scene_id, sequence, title, description, image, video_url, "
        "location, time_of_day, characters, mood, is_ai_generated, is_subdivided, video_segments_json, middle_frame_urls, "
        "context_snapshot, created_at, updated_at, deleted_at, camera_angle, lighting, color_palette) VALUES"
    )
    lines.append(",\n".join(scv) + ";")
    lines.append("")

    # 7) Fragments (35)
    fv = []
    topics = ("writing", "scifi", "fantasy", "daily", "idea")
    for i, fid in enumerate(frags, start=1):
        cid = users[i % nu]
        fv.append(
            f"('{fid}', '{cid}', 'Fragment body text {i} for feed testing.', '[]', 'public', 'original', '', "
            f"'{topics[i % 5]}', 'Caption {i}', NULL, 0, {i % 20}, {i % 5}, 0, {i * 2}, "
            f"UNIX_TIMESTAMP(), UNIX_TIMESTAMP())"
        )
    lines.append(
        "INSERT INTO fragments (id, creator_id, content, image_urls, visibility, source_type, source_id, topic, caption, "
        "converted_to_story_id, is_converted, likes, comments, shares, views, created_at, updated_at) VALUES"
    )
    lines.append(",\n".join(fv) + ";")
    lines.append("")

    # 8) Tags (10)
    tv = []
    names = ["seed-tag-a", "seed-tag-b", "seed-tag-c", "seed-adventure", "seed-romance", "seed-mystery", "seed-ai", "seed-short", "seed-long", "seed-nsfw"]
    for t, name in zip(tags, names):
        tv.append(f"('{t}', '{name}', 'genre', 0, NOW(3), NULL)")
    lines.append("INSERT INTO tags (id, name, category, usage_count, created_at, deleted_at) VALUES")
    lines.append(",\n".join(tv) + ";")
    lines.append("")

    # 9) Story tags (25)
    stgv = []
    for i, sid in enumerate(stories, start=1):
        tid = tags[i % 10]
        stgv.append(f"(UUID(), '{sid}', '{tid}', NOW(3), NULL)")
    lines.append("INSERT INTO story_tags (id, story_id, tag_id, created_at, deleted_at) VALUES")
    lines.append(",\n".join(stgv) + ";")
    lines.append("")

    # 10) Story likes (25)
    slv = []
    for i, sid in enumerate(stories, start=1):
        uid_ = users[i % nu]
        slv.append(f"(UUID(), '{uid_}', '{sid}', NOW(3), NULL)")
    lines.append("INSERT INTO story_likes (id, user_id, story_id, created_at, deleted_at) VALUES")
    lines.append(",\n".join(slv) + ";")
    lines.append("")

    # 11) Storyboard likes (25)
    sblv = []
    for i, bid in enumerate(boards, start=1):
        uid_ = users[(i + 3) % nu]
        sblv.append(f"(UUID(), '{uid_}', '{bid}', NOW(3), NULL)")
    lines.append("INSERT INTO storyboard_likes (id, user_id, storyboard_id, created_at, deleted_at) VALUES")
    lines.append(",\n".join(sblv) + ";")
    lines.append("")

    # 12) Polymorphic likes on fragments (20)
    plv = []
    for i in range(20):
        fid = frags[i % nf]
        uid_ = users[(i + 1) % nu]
        plv.append(f"(UUID(), '{uid_}', 'fragment', '{fid}', UNIX_TIMESTAMP())")
    lines.append("INSERT INTO likes (id, user_id, likeable_type, likeable_id, created_at) VALUES")
    lines.append(",\n".join(plv) + ";")
    lines.append("")

    # 13) Bookmarks (20)
    bmv = []
    for i in range(20):
        uid_ = users[i % nu]
        if i % 3 == 0:
            btype, bid = "story", stories[i % ns]
        elif i % 3 == 1:
            btype, bid = "fragment", frags[i % nf]
        else:
            btype, bid = "storyboard", boards[i % ns]
        bmv.append(f"(UUID(), '{uid_}', '{btype}', '{bid}', NULL, NOW(3))")
    lines.append("INSERT INTO bookmarks (id, user_id, bookmark_type, bookmark_id, collection_name, created_at) VALUES")
    lines.append(",\n".join(bmv) + ";")
    lines.append("")

    # 14) Fragment likes (20)
    flv = []
    for i in range(20):
        fid = frags[(i + 5) % nf]
        uid_ = users[(i + 2) % nu]
        flv.append(f"(UUID(), '{fid}', '{uid_}', UNIX_TIMESTAMP())")
    lines.append("INSERT INTO fragment_likes (id, fragment_id, user_id, created_at) VALUES")
    lines.append(",\n".join(flv) + ";")
    lines.append("")

    # 15) Fragment comments (15)
    fcv = []
    for i in range(15):
        fid = frags[i % nf]
        uid_ = users[(i + 7) % nu]
        fcv.append(
            f"(UUID(), '{fid}', '{uid_}', 'Comment {i} on fragment', NULL, UNIX_TIMESTAMP(), UNIX_TIMESTAMP())"
        )
    lines.append(
        "INSERT INTO fragment_comments (id, fragment_id, user_id, content, parent_id, created_at, updated_at) VALUES"
    )
    lines.append(",\n".join(fcv) + ";")
    lines.append("")

    # 16) User follows (15)
    ufv = []
    for i in range(15):
        a, b = users[i % nu], users[(i + 1) % nu]
        ufv.append(f"(UUID(), '{a}', '{b}', NOW(3), NULL)")
    lines.append("INSERT INTO user_follows (id, follower_id, followee_id, created_at, deleted_at) VALUES")
    lines.append(",\n".join(ufv) + ";")
    lines.append("")

    # 17) Story follows (12)
    sfv = []
    for i in range(12):
        uid_ = users[(i + 4) % nu]
        sid = stories[(i + 2) % ns]
        sfv.append(f"(UUID(), '{uid_}', '{sid}', NOW(3), NULL)")
    lines.append("INSERT INTO story_follows (id, user_id, story_id, created_at, deleted_at) VALUES")
    lines.append(",\n".join(sfv) + ";")
    lines.append("")

    # 18) View histories (20)
    vhv = []
    for i in range(20):
        uid_ = users[i % nu]
        if i % 2 == 0:
            et, eid = "story", stories[i % ns]
        else:
            et, eid = "fragment", frags[i % nf]
        vhv.append(f"(UUID(), '{uid_}', '{et}', '{eid}', {30 + i}, NOW(3))")
    lines.append("INSERT INTO view_histories (id, user_id, entity_type, entity_id, duration, viewed_at) VALUES")
    lines.append(",\n".join(vhv) + ";")
    lines.append("")

    # 19) Search histories (15)
    shv = []
    queries = ["dragon", "space", "love", "mystery", "seed"]
    for i in range(15):
        uid_ = users[i % nu]
        q = queries[i % 5]
        shv.append(f"(UUID(), '{uid_}', '{q} {i}', 'story', {3 + i % 10}, NOW(3))")
    lines.append("INSERT INTO search_histories (id, user_id, query, type, result_count, created_at) VALUES")
    lines.append(",\n".join(shv) + ";")
    lines.append("")

    # 20) Notifications (15)
    nv = []
    for i in range(15):
        uid_ = users[(i + 1) % nu]
        actor = users[i % nu]
        sid = stories[i % ns]
        nv.append(
            f"(UUID(), '{uid_}', 'like', 'Seed notify {i}', 'Someone liked your story', '/story/{sid}', 0, "
            f"'{actor}', 'Actor', NULL, 'Seed Story', NULL, '{sid}', NULL, NULL, NULL, NULL, NOW(3), NULL)"
        )
    lines.append(
        "INSERT INTO notifications (id, user_id, type, title, content, link, `read`, actor_id, actor_name, actor_avatar, "
        "story_title, story_cover, story_id, comment_text, sys_title, sys_body, sys_icon, created_at, deleted_at) VALUES"
    )
    lines.append(",\n".join(nv) + ";")
    lines.append("")

    lines.append("SET FOREIGN_KEY_CHECKS = 1;")
    lines.append("")
    # Row count note
    total = 20 * 2 + 25 * 5 + 35 + 10 + 25 + 25 + 25 + 20 + 20 + 20 + 15 + 15 + 12 + 20 + 15 + 15
    lines.append(f"-- Approx row inserts above: {total} (target >= 200)")

    OUT.write_text("\n".join(lines), encoding="utf-8")
    print(f"Wrote {OUT} ({total} rows)")


if __name__ == "__main__":
    main()
