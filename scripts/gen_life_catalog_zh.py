#!/usr/bin/env python3
"""Generate pkg/i18n/locales/life_catalog.zh.toml from Life seed catalogs."""

from __future__ import annotations

import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "pkg" / "i18n" / "locales" / "life_catalog.zh.toml"

SKILLS: dict[str, dict[str, str]] = {
    "int": {"title": "智力", "subtitle": "研究、分析与学习循环"},
    "int-research": {"title": "研究"},
    "int-analysis": {"title": "分析"},
    "int-systems": {"title": "系统"},
    "wri": {"title": "写作", "subtitle": "起草、文档与发布"},
    "wri-draft": {"title": "起草"},
    "wri-docs": {"title": "文档"},
    "wri-story": {"title": "叙事"},
    "foc": {"title": "专注", "subtitle": "规划、深度工作与跟进"},
    "foc-plan": {"title": "规划"},
    "foc-exec": {"title": "执行"},
    "foc-deep": {"title": "深度工作"},
    "cre": {"title": "创造力", "subtitle": "构思、原型与制作"},
    "cre-ideas": {"title": "构思"},
    "cre-design": {"title": "设计"},
    "cre-proto": {"title": "原型"},
    "cha": {"title": "魅力", "subtitle": "沟通、教学与领导"},
    "cha-comm": {"title": "沟通"},
    "cha-lead": {"title": "领导"},
    "cha-teach": {"title": "教学"},
    "phy": {"title": "体魄", "subtitle": "训练、灵活与恢复"},
    "phy-strength": {"title": "力量"},
    "phy-endurance": {"title": "耐力"},
    "phy-recovery": {"title": "恢复"},
    "wil": {"title": "意志", "subtitle": "纪律与一致性习惯"},
    "wil-discipline": {"title": "纪律"},
    "wil-consistency": {"title": "一致性"},
    "wil-recovery": {"title": "韧性"},
    "fin": {"title": "财务", "subtitle": "预算、销售与谈判"},
    "fin-budget": {"title": "预算"},
    "fin-sales": {"title": "销售"},
    "fin-negotiation": {"title": "谈判"},
    "general": {"title": "通用", "subtitle": "未映射但仍值得关注的成长信号"},
    "general-explore": {"title": "探索"},
}

ACHIEVEMENTS: dict[str, dict[str, str]] = {
    "ach-first-quest": {"name": "初出茅庐", "description": "告别教程——完成第一个任务。"},
    "ach-quests-7": {"name": "一周行动", "description": "完成 7 个任务。动量从小处开始。"},
    "ach-quests-21": {"name": "三周节奏", "description": "完成 21 个任务。久到像习惯。"},
    "ach-quests-50": {"name": "老练冒险者", "description": "完成 50 个任务。你不再只是浅尝辄止。"},
    "ach-quests-100": {"name": "百项里程碑", "description": "完成 100 个任务。扎实的工作体量。"},
    "ach-quests-200": {"name": "长期战役", "description": "完成 200 个任务。数月的坚持。"},
    "ach-quests-365": {"name": "任务之年", "description": "完成 365 个任务。一整年的前进——按你的节奏。"},
    "ach-first-daily": {"name": "日常节奏", "description": "完成第一个日常任务。循环从这里开始。"},
    "ach-daily-7": {"name": "七次回归", "description": "完成 7 个日常任务。一周的练习。"},
    "ach-daily-21": {"name": "习惯成形", "description": "完成 21 个日常任务。习惯的 groove 正在形成。"},
    "ach-daily-66": {"name": "身份转变", "description": "完成 66 个日常任务。越过经典习惯养成窗口。"},
    "ach-daily-100": {"name": "百日", "description": "完成 100 个日常任务。平静的一致。"},
    "ach-daily-200": {"name": "不间断练习", "description": "完成 200 个日常任务。练习已成为你的一部分。"},
    "ach-first-one-time": {"name": "一击", "description": "完成第一个一次性任务。关闭一个真实循环。"},
    "ach-one-time-5": {"name": "收尾五件", "description": "完成 5 个一次性任务。小闭环会叠加。"},
    "ach-one-time-12": {"name": "项目终结者", "description": "完成 12 个一次性任务。十二条已完结的线。"},
    "ach-one-time-30": {"name": "交付清单", "description": "完成 30 个一次性任务。可见的完成轨迹。"},
    "ach-one-time-60": {"name": "档案建造者", "description": "完成 60 个一次性任务。你的档案越来越厚。"},
    "ach-one-time-100": {"name": "百次闭环", "description": "完成 100 个一次性任务。说到做到。"},
    "ach-first-boss": {"name": "Boss 通关", "description": "完成第一个 Boss 任务。真正的里程碑，不是杂活。"},
    "ach-boss-3": {"name": "团本学徒", "description": "完成 3 个 Boss 任务。你知道大项目是什么感觉。"},
    "ach-boss-7": {"name": "团本赛季", "description": "完成 7 个 Boss 任务。一个艰难通关的赛季。"},
    "ach-boss-15": {"name": "团本队长", "description": "完成 15 个 Boss 任务。稀有工作，反复完成。"},
    "ach-boss-30": {"name": "团本传奇", "description": "完成 30 个 Boss 任务。以艰难收官为职业。"},
    "ach-first-b": {"name": "B 级通关", "description": "完成 B 级任务。真实努力，不是热身。"},
    "ach-first-a": {"name": "A 级通关", "description": "完成 A 级任务。功能级工作，已交付。"},
    "ach-first-s": {"name": "S 级通关", "description": "完成 S 级任务。高风险，干净收尾。"},
    "ach-first-ss": {"name": "双 S", "description": "完成 SS 级任务。接近神话的难度。"},
    "ach-first-sss": {"name": "神话通关", "description": "完成 SSS 级任务。发布级稀有度。"},
    "ach-b-10": {"name": "B 级十连", "description": "完成 10 个 B 级任务。扎实工作，反复出现。"},
    "ach-a-5": {"name": "A 级五连", "description": "完成 5 个 A 级任务。五个已上线的功能。"},
    "ach-a-15": {"name": "A 级工匠", "description": "完成 15 个 A 级任务。工艺胜过运气。"},
    "ach-s-3": {"name": "S 级三连", "description": "完成 3 个 S 级任务。三次艰难胜利。"},
    "ach-s-10": {"name": "S 级十连", "description": "完成 10 个 S 级任务。高门槛，习惯性达成。"},
    "ach-ss-3": {"name": "SS 三连", "description": "完成 3 个 SS 级任务。三次接近神话的通关。"},
    "ach-sss-3": {"name": "神话三连", "description": "完成 3 个 SSS 级任务。三次发布级收官。"},
    "ach-daily-first-b": {"name": "困难日常", "description": "完成 B 级日常任务。仍有拉伸的日常。"},
    "ach-daily-first-a": {"name": "深度练习", "description": "完成 A 级日常任务。功能深度的重复工作。"},
    "ach-daily-a-5": {"name": "深度练习五连", "description": "完成 5 个 A 级日常任务。困难习惯，保持住。"},
    "ach-onetime-first-a": {"name": "功能完成", "description": "完成 A 级一次性任务。一个真实功能，已关闭。"},
    "ach-onetime-a-5": {"name": "功能连击", "description": "完成 5 个 A 级一次性任务。五个离开看板的功能。"},
    "ach-onetime-first-s": {"name": "高风险收尾", "description": "完成 S 级一次性任务。一条难线，已完结。"},
    "ach-boss-first-s": {"name": "Boss S 通关", "description": "完成 S 级 Boss 任务。压力下的里程碑。"},
    "ach-boss-first-ss": {"name": "Boss SS 通关", "description": "完成 SS 级 Boss 任务。接近神话的团本通关。"},
    "ach-boss-first-sss": {"name": "Boss 神话", "description": "完成 SSS 级 Boss 任务。Life 中最稀有的通关。"},
    "ach-boss-ss-3": {"name": "SS 团本三连", "description": "完成 3 个 SS 级 Boss 任务。三次接近神话的团本。"},
    "ach-boss-sss-3": {"name": "神话团本三连", "description": "完成 3 个 SSS 级 Boss 任务。三次发布级团本。"},
}


EQUIP_LORE_ZH: dict[str, str] = {
    "Soft armor against context-switch chill.": "抵御切换上下文寒意的柔软护甲。",
}

WORD_ZH: dict[str, str] = {
    "Focus": "专注",
    "Spectacles": "眼镜",
    "Mithril": "秘银",
    "Keyboard": "键盘",
    "Steady": "沉稳",
    "Hoodie": "连帽衫",
    "Perpetual": "永动",
    "Sneakers": "运动鞋",
    "Charm": "魅力",
    "Token": "信物",
    "Ring": "戒指",
    "Truth": "真理",
    "Codex": "法典",
    "Legend": "传奇",
    "Quill": "羽笔",
    "Mythic": "神话",
    "Crown": "王冠",
    "Clarity": "清晰",
    "Canvas": "画布",
    "Cap": "帽",
    "Worn": "旧",
    "Knit": "针织",
    "Soft": "柔软",
    "Beanie": "毛线帽",
    "Secondhand": "二手",
    "Noise": "降噪",
    "Band": "带",
    "Desk": "桌面",
    "Visor": "遮阳",
    "Morning": "晨间",
    "Cloth": "布",
    "Headband": "头带",
    "Reading": "阅读",
    "Clip": "夹",
    "Draft": "草稿",
    "Pen": "笔",
    "Budget": "预算",
    "Mouse": "鼠标",
    "Sticky": "便利",
    "Note": "贴",
    "Pad": "垫",
    "Office": "办公",
    "Mug": "杯",
    "Basic": "基础",
    "Stylus": "触控笔",
    "Timer": "计时",
    "Clicker": "器",
    "Kitchen": "厨房",
    "Linen": "亚麻",
    "Work": "工作",
    "Shirt": "衬衫",
    "Armor": "护甲",
    "Weapon": "武器",
    "Boots": "靴",
    "Shoes": "鞋",
    "Ancient": "远古",
    "Blessed": "祝福",
    "Broken": "破损",
    "Bright": "明亮",
    "Calm": "平静",
    "Carbon": "碳纤",
    "Chain": "链",
    "Crystal": "水晶",
    "Deep": "深",
    "Digital": "数字",
    "Echo": "回响",
    "Field": "野",
    "Glass": "玻璃",
    "Gold": "金",
    "Guard": "守卫",
    "Iron": "铁",
    "Light": "光",
    "Moon": "月",
    "Night": "夜",
    "Old": "旧",
    "Quick": "快",
    "Quiet": "静",
    "River": "河",
    "Run": "跑",
    "Shadow": "影",
    "Silver": "银",
    "Sky": "天",
    "Stone": "石",
    "Storm": "风暴",
    "Sun": "日",
    "Swift": "迅",
    "Travel": "旅行",
    "Warm": "暖",
    "Wind": "风",
    "Winter": "冬",
    "Wood": "木",
    "World": "世界",
    "Young": "年轻",
    "of": "之",
    "the": "",
    "and": "与",
    "a": "",
    "an": "",
    "for": "为",
    "in": "于",
}


def escape_toml(s: str) -> str:
    return s.replace("\\", "\\\\").replace('"', '\\"')


def translate_equip_name(name: str) -> str:
    words = re.findall(r"[A-Za-z0-9]+", name)
    if not words:
        return name
    parts: list[str] = []
    for word in words:
        if word in WORD_ZH:
            zh = WORD_ZH[word]
            if zh:
                parts.append(zh)
            continue
        subs = re.findall(r"[A-Z]?[a-z]+|[A-Z]+(?=[A-Z][a-z]|\b)", word)
        if len(subs) > 1:
            chunk = "".join(WORD_ZH.get(sub, sub) for sub in subs)
            parts.append(chunk)
        else:
            parts.append(WORD_ZH.get(word, word))
    translated = "".join(parts)
    return translated if translated.strip() else name


def write_section(lines: list[str], key: str, value: str) -> None:
    lines.append(f"[{key}]")
    lines.append(f'other = "{escape_toml(value)}"')
    lines.append("")


def main() -> None:
    lines: list[str] = [
        "# Life catalog copy (achievements, skill tree, equipment). Generated by scripts/gen_life_catalog_zh.py",
        "",
    ]

    for key, fields in sorted(SKILLS.items()):
        if "title" in fields:
            write_section(lines, f"life.skill.{key}.title", fields["title"])
        if fields.get("subtitle"):
            write_section(lines, f"life.skill.{key}.subtitle", fields["subtitle"])

    ach_path = ROOT / "internal" / "modules" / "life" / "seed" / "achievements.json"
    achievements = json.loads(ach_path.read_text(encoding="utf-8"))
    for item in achievements:
        flag = item["flag"]
        zh = ACHIEVEMENTS.get(flag, {})
        name = zh.get("name", item["name"])
        desc = zh.get("description", item["description"])
        write_section(lines, f"life.ach.{flag}.name", name)
        write_section(lines, f"life.ach.{flag}.description", desc)

    eq_path = ROOT / "internal" / "modules" / "life" / "seed" / "equipment.json"
    equipment = json.loads(eq_path.read_text(encoding="utf-8"))
    for item in equipment:
        flag = item["flag"]
        write_section(lines, f"life.equip.{flag}.name", translate_equip_name(item["name"]))
        lore = item.get("ai_lore_text") or ""
        lore_zh = EQUIP_LORE_ZH.get(lore)
        if lore_zh:
            write_section(lines, f"life.equip.{flag}.lore", lore_zh)

    OUT.write_text("\n".join(lines) + "\n", encoding="utf-8")
    print(f"wrote {OUT} ({len(lines)} lines)")


if __name__ == "__main__":
    main()
