from __future__ import annotations

from pathlib import Path
from textwrap import dedent

from PIL import Image, ImageDraw, ImageFont


ROOT = Path(__file__).resolve().parents[1]
ASSETS = ROOT / "assets"
LOGO = ASSETS / "logo"
FAVICON = ASSETS / "favicon"
SOCIAL = ASSETS / "social"

FONT_STACK = "'Segoe UI', 'Arial', sans-serif"
FONT_BOLD = [
    Path("C:/Windows/Fonts/segoeuib.ttf"),
    Path("C:/Windows/Fonts/arialbd.ttf"),
]
FONT_REGULAR = [
    Path("C:/Windows/Fonts/segoeui.ttf"),
    Path("C:/Windows/Fonts/arial.ttf"),
]


def main() -> None:
    for directory in (LOGO, FAVICON, SOCIAL):
        directory.mkdir(parents=True, exist_ok=True)

    (LOGO / "logo.svg").write_text(build_logo_svg(theme="light", background=False, width=1200, height=420), encoding="utf-8")

    draw_logo_png(LOGO / "logo-light.png", theme="light", width=1400, height=490, background=False)
    draw_logo_png(LOGO / "logo-dark.png", theme="dark", width=1400, height=490, background=True)
    draw_symbol_png(LOGO / "symbol-light.png", theme="light", size=640, background=False)
    draw_symbol_png(LOGO / "symbol-dark.png", theme="dark", size=640, background=True)
    draw_monogram_png(LOGO / "monogram.png", size=640)
    draw_social_png(SOCIAL / "github-social-preview.png", width=1280, height=640)
    draw_favicon_png(FAVICON / "favicon-32.png", size=32)
    draw_favicon_png(FAVICON / "favicon-16.png", size=16)

    favicon_img = Image.open(FAVICON / "favicon-32.png")
    favicon_img.save(FAVICON / "favicon.ico", sizes=[(16, 16), (32, 32)])


def palette(theme: str) -> dict[str, tuple[int, int, int, int]]:
    if theme == "dark":
        return {
            "bg": (7, 26, 51, 255),
            "panel": (17, 45, 82, 255),
            "text": (248, 251, 255, 255),
            "subtext": (215, 232, 251, 255),
            "border": (22, 52, 89, 255),
            "gopher": (76, 188, 245, 255),
            "gopher_hi": (110, 214, 255, 255),
            "wave": (31, 163, 242, 255),
            "navy": (10, 31, 68, 255),
            "shield": (11, 35, 76, 255),
            "white": (248, 251, 255, 255),
            "accent": (31, 163, 242, 255),
        }
    return {
        "bg": (255, 255, 255, 0),
        "panel": (255, 255, 255, 0),
        "text": (10, 31, 68, 255),
        "subtext": (31, 59, 103, 255),
        "border": (255, 255, 255, 0),
        "gopher": (76, 188, 245, 255),
        "gopher_hi": (110, 214, 255, 255),
        "wave": (31, 163, 242, 255),
        "navy": (10, 31, 68, 255),
        "shield": (11, 35, 76, 255),
        "white": (248, 251, 255, 255),
        "accent": (31, 163, 242, 255),
    }


def font(size: int, bold: bool = False) -> ImageFont.FreeTypeFont | ImageFont.ImageFont:
    candidates = FONT_BOLD if bold else FONT_REGULAR
    for candidate in candidates:
        if candidate.exists():
            return ImageFont.truetype(str(candidate), size=size)
    return ImageFont.load_default()


def new_canvas(width: int, height: int, bg: tuple[int, int, int, int]) -> tuple[Image.Image, ImageDraw.ImageDraw]:
    image = Image.new("RGBA", (width, height), bg)
    return image, ImageDraw.Draw(image)


def draw_logo_png(path: Path, theme: str, width: int, height: int, background: bool) -> None:
    colors = palette(theme)
    bg = colors["bg"] if background else (255, 255, 255, 0)
    image, draw = new_canvas(width, height, bg)

    if background:
        draw.rounded_rectangle((2, 2, width - 2, height - 2), radius=28, fill=colors["bg"], outline=colors["border"], width=2)

    draw_symbol(draw, center=(250, 215), scale=0.92, theme=theme)

    title_font = font(146, bold=True)
    subtitle_font = font(34, bold=True)
    ws_width = draw.textbbox((0, 0), "WS", font=title_font)[2]
    auth_width = draw.textbbox((0, 0), "Auth", font=title_font)[2]

    x = 420
    y = 118
    draw.text((x, y), "WS", font=title_font, fill=colors["text"])
    draw.text((x + ws_width - 4, y), "Auth", font=title_font, fill=colors["accent"])
    draw.text((x + ws_width + auth_width - 8, y), "Kit", font=title_font, fill=colors["text"])

    line_y = 344
    draw.line((430, line_y, 540, line_y), fill=colors["wave"], width=6)
    draw.line((1088, line_y, 1198, line_y), fill=colors["wave"], width=6)
    draw.text((565, 320), "SECURE WEBSOCKET AUTHENTICATION FOR GO", font=subtitle_font, fill=colors["subtext"], anchor="la")

    image.save(path)


def draw_symbol_png(path: Path, theme: str, size: int, background: bool) -> None:
    colors = palette(theme)
    bg = colors["bg"] if background else (255, 255, 255, 0)
    image, draw = new_canvas(size, size, bg)
    if background:
        draw.rounded_rectangle((2, 2, size - 2, size - 2), radius=64, fill=colors["bg"], outline=colors["border"], width=2)
    draw_symbol(draw, center=(size / 2, size / 2), scale=size / 512, theme=theme)
    image.save(path)


def draw_monogram_png(path: Path, size: int) -> None:
    image, draw = new_canvas(size, size, (255, 255, 255, 255))
    navy = (10, 31, 68, 255)
    accent = (31, 163, 242, 255)
    draw.ellipse((80, 80, size - 80, size - 80), outline=navy, width=12)
    draw.arc((84, 84, size - 84, size - 84), start=205, end=300, fill=accent, width=18)
    draw.arc((84, 84, size - 84, size - 84), start=25, end=118, fill=accent, width=18)
    draw.ellipse((72, size / 2 - 10, 92, size / 2 + 10), fill=accent)
    draw.ellipse((size - 92, size / 2 - 10, size - 72, size / 2 + 10), fill=accent)
    mono_font = font(180, bold=True)
    draw.text((size / 2, size / 2 + 12), "WS", font=mono_font, fill=navy, anchor="mm")
    accent_font = font(180, bold=True)
    draw.text((size / 2 + 62, size / 2 + 12), "S", font=accent_font, fill=accent, anchor="mm")
    image.save(path)


def draw_social_png(path: Path, width: int, height: int) -> None:
    colors = palette("dark")
    image, draw = new_canvas(width, height, colors["bg"])
    draw.rounded_rectangle((0, 0, width, height), radius=40, fill=colors["bg"])
    draw.ellipse((920, -60, 1290, 300), fill=(20, 54, 99, 90))
    draw.ellipse((860, 360, 1320, 760), fill=(14, 41, 77, 100))

    draw_symbol(draw, center=(255, 265), scale=1.22, theme="dark")

    title_font = font(146, bold=True)
    sub_font = font(34, bold=True)
    body_font = font(24, bold=False)

    ws_width = draw.textbbox((0, 0), "WS", font=title_font)[2]
    auth_width = draw.textbbox((0, 0), "Auth", font=title_font)[2]
    x = 425
    y = 155
    draw.text((x, y), "WS", font=title_font, fill=colors["text"])
    draw.text((x + ws_width - 3, y), "Auth", font=title_font, fill=colors["accent"])
    draw.text((x + ws_width + auth_width - 6, y), "Kit", font=title_font, fill=colors["text"])
    draw.text((430, 320), "SECURE WEBSOCKET AUTHENTICATION FOR GO", font=sub_font, fill=colors["subtext"])
    draw.text((430, 404), "JWT authentication middleware for cloud-native websocket services.", font=body_font, fill=(175, 203, 231, 255))
    draw.text((430, 446), "Header and subprotocol extraction. Issuer and audience validation.", font=body_font, fill=(175, 203, 231, 255))
    draw.text((430, 488), "Clean handlers with secure defaults.", font=body_font, fill=(175, 203, 231, 255))
    image.save(path)


def draw_favicon_png(path: Path, size: int) -> None:
    image, draw = new_canvas(size, size, (20, 153, 231, 255))
    draw.rounded_rectangle((0, 0, size, size), radius=max(6, size // 5), fill=(20, 153, 231, 255))
    draw_symbol(draw, center=(size / 2, size / 2), scale=size / 640, theme="dark")
    image.save(path)


def draw_symbol(draw: ImageDraw.ImageDraw, center: tuple[float, float], scale: float, theme: str) -> None:
    colors = palette(theme)
    cx, cy = center

    def t(point: tuple[float, float]) -> tuple[float, float]:
        x, y = point
        return cx + (x - 128) * scale, cy + (y - 128) * scale

    def box(x1: float, y1: float, x2: float, y2: float) -> tuple[float, float, float, float]:
        p1 = t((x1, y1))
        p2 = t((x2, y2))
        return p1[0], p1[1], p2[0], p2[1]

    width = max(2, int(10 * scale))

    draw.arc(box(-18, 16, 70, 160), start=126, end=232, fill=colors["wave"], width=width)
    draw.arc(box(2, 44, 82, 154), start=126, end=228, fill=colors["wave"], width=width)
    draw.arc(box(186, 16, 274, 160), start=-52, end=54, fill=colors["wave"], width=width)
    draw.arc(box(174, 44, 254, 154), start=-48, end=54, fill=colors["wave"], width=width)
    r = max(4, int(7 * scale))
    dot_l = t((20, 94))
    dot_r = t((236, 94))
    draw.ellipse((dot_l[0] - r, dot_l[1] - r, dot_l[0] + r, dot_l[1] + r), fill=colors["wave"])
    draw.ellipse((dot_r[0] - r, dot_r[1] - r, dot_r[0] + r, dot_r[1] + r), fill=colors["wave"])

    body = box(76, 24, 180, 194)
    draw.rounded_rectangle(body, radius=52 * scale, fill=colors["gopher"], outline=colors["navy"], width=max(2, int(4 * scale)))
    for ear in ((96, 36), (160, 36)):
        ex, ey = t(ear)
        er = 16 * scale
        draw.ellipse((ex - er, ey - er, ex + er, ey + er), fill=colors["gopher"], outline=colors["navy"], width=max(2, int(4 * scale)))

    for eye in ((108, 84), (148, 84)):
        ex, ey = t(eye)
        er = 26 * scale
        pr = 8 * scale
        draw.ellipse((ex - er, ey - er, ex + er, ey + er), fill=colors["white"], outline=colors["navy"], width=max(2, int(4 * scale)))
        draw.ellipse((ex - pr, ey - pr, ex + pr, ey + pr), fill=colors["navy"])

    draw.arc(box(118, 96, 138, 112), start=200, end=340, fill=colors["navy"], width=max(2, int(4 * scale)))
    nose = box(118, 104, 138, 122)
    draw.rounded_rectangle(nose, radius=8 * scale, fill=colors["white"], outline=colors["navy"], width=max(2, int(4 * scale)))
    draw.line((t((126, 122)), t((126, 136))), fill=colors["navy"], width=max(2, int(4 * scale)))
    draw.line((t((130, 122)), t((130, 138))), fill=colors["navy"], width=max(2, int(4 * scale)))

    shield_outline = [t((80, 112)), t((128, 96)), t((176, 112)), t((168, 176)), t((128, 204)), t((88, 176))]
    shield_fill = [t((94, 118)), t((128, 107)), t((162, 118)), t((156, 170)), t((128, 191)), t((100, 170))]
    draw.polygon(shield_outline, fill=colors["white"], outline=colors["navy"])
    draw.polygon(shield_fill, fill=colors["shield"])

    lock_box = box(114, 132, 142, 160)
    draw.rounded_rectangle(lock_box, radius=6 * scale, fill=colors["white"])
    shackle = [
        t((118, 132)),
        t((118, 124)),
        t((121, 114)),
        t((128, 110)),
        t((135, 114)),
        t((138, 124)),
        t((138, 132)),
    ]
    draw.line(shackle, fill=colors["white"], width=max(3, int(6 * scale)), joint="curve")
    key_cx, key_cy = t((128, 146))
    kr = max(2, int(4 * scale))
    draw.ellipse((key_cx - kr, key_cy - kr, key_cx + kr, key_cy + kr), fill=colors["navy"])
    draw.rounded_rectangle((key_cx - kr / 2, key_cy, key_cx + kr / 2, key_cy + 8 * scale), radius=2 * scale, fill=colors["navy"])


def build_logo_svg(theme: str, background: bool, width: int, height: int) -> str:
    bg_fill = "#FFFFFF" if theme == "light" else "#071A33"
    border = "none" if theme == "light" else "#163459"
    text_primary = "#0A1F44" if theme == "light" else "#F8FBFF"
    accent = "#1E9AE6"
    subtext = "#1F3B67" if theme == "light" else "#D7E8FB"
    wave = "#1FA3F2"
    symbol = build_symbol_group(210, 210, 0.78)
    bg = f"<rect width='{width}' height='{height}' rx='28' fill='{bg_fill}' stroke='{border}' stroke-width='2'/>" if background else ""
    return dedent(
        f"""
        <svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" viewBox="0 0 {width} {height}">
          {bg}
          {symbol}
          <text x="355" y="220" font-family="{FONT_STACK}" font-size="102" font-weight="800" letter-spacing="-2">
            <tspan fill="{text_primary}">WS</tspan><tspan fill="{accent}">Auth</tspan><tspan fill="{text_primary}">Kit</tspan>
          </text>
          <line x1="360" x2="458" y1="305" y2="305" stroke="{wave}" stroke-width="6" stroke-linecap="round"/>
          <line x1="900" x2="998" y1="305" y2="305" stroke="{wave}" stroke-width="6" stroke-linecap="round"/>
          <text x="480" y="318" font-family="{FONT_STACK}" font-size="26" font-weight="700" letter-spacing="2.5" fill="{subtext}">
            SECURE WEBSOCKET AUTHENTICATION FOR GO
          </text>
        </svg>
        """
    ).strip()


def build_symbol_group(cx: float, cy: float, scale: float) -> str:
    return dedent(
        f"""
        <g transform="translate({cx} {cy}) scale({scale}) translate(-128 -128)">
          <path d="M42 136 C10 110, 6 68, 22 34" fill="none" stroke="#1FA3F2" stroke-width="10" stroke-linecap="round"/>
          <path d="M56 150 C28 128, 22 92, 34 60" fill="none" stroke="#1FA3F2" stroke-width="10" stroke-linecap="round"/>
          <path d="M214 136 C246 110, 250 68, 234 34" fill="none" stroke="#1FA3F2" stroke-width="10" stroke-linecap="round"/>
          <path d="M200 150 C228 128, 234 92, 222 60" fill="none" stroke="#1FA3F2" stroke-width="10" stroke-linecap="round"/>
          <circle cx="20" cy="94" r="7" fill="#1FA3F2"/>
          <circle cx="236" cy="94" r="7" fill="#1FA3F2"/>
          <rect x="76" y="24" width="104" height="170" rx="52" fill="#4CBCF5" stroke="#0A1F44" stroke-width="4"/>
          <circle cx="96" cy="36" r="16" fill="#4CBCF5" stroke="#0A1F44" stroke-width="4"/>
          <circle cx="160" cy="36" r="16" fill="#4CBCF5" stroke="#0A1F44" stroke-width="4"/>
          <circle cx="108" cy="84" r="26" fill="#F8FBFF" stroke="#0A1F44" stroke-width="4"/>
          <circle cx="148" cy="84" r="26" fill="#F8FBFF" stroke="#0A1F44" stroke-width="4"/>
          <circle cx="114" cy="84" r="8" fill="#0A1F44"/>
          <circle cx="154" cy="84" r="8" fill="#0A1F44"/>
          <path d="M118 104 C124 98, 132 98, 138 104" fill="none" stroke="#0A1F44" stroke-width="4" stroke-linecap="round"/>
          <rect x="118" y="104" width="20" height="18" rx="8" fill="#F8FBFF" stroke="#0A1F44" stroke-width="4"/>
          <line x1="126" y1="122" x2="126" y2="136" stroke="#0A1F44" stroke-width="4" stroke-linecap="round"/>
          <line x1="130" y1="122" x2="130" y2="138" stroke="#0A1F44" stroke-width="4" stroke-linecap="round"/>
          <path d="M80 112 L128 96 L176 112 L168 176 L128 204 L88 176 Z" fill="#F8FBFF" stroke="#0A1F44" stroke-width="4"/>
          <path d="M94 118 L128 107 L162 118 L156 170 L128 191 L100 170 Z" fill="#0B234C"/>
          <rect x="114" y="132" width="28" height="28" rx="6" fill="#F8FBFF"/>
          <path d="M118 132 V124 C118 116, 123 110, 128 110 C133 110, 138 116, 138 124 V132" fill="none" stroke="#F8FBFF" stroke-width="6" stroke-linecap="round"/>
          <circle cx="128" cy="146" r="4" fill="#0A1F44"/>
          <rect x="126" y="146" width="4" height="8" rx="2" fill="#0A1F44"/>
        </g>
        """
    ).strip()


if __name__ == "__main__":
    main()
