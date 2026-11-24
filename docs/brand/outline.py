"""nex-ovia mark explorations. The oval is the O of nex-o-via, parted by a horizontal cut;
'nex' owns the top half, 'via' the bottom. Glyphs are outlined (no font needed)."""
import sys, math
from fontTools.ttLib import TTFont
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.pens.transformPen import TransformPen
from fontTools.pens.boundsPen import BoundsPen

FD = __import__("os").path.join(__import__("os").path.dirname(__file__), "font")
fonts = {w: TTFont(f"{FD}/Unbounded-{w}.ttf") for w in (300, 800, 900)}

def outline(text, w, size, track=0):
    f = fonts[w]; gs = f.getGlyphSet(); cmap = f.getBestCmap(); s = size / f['head'].unitsPerEm
    names = [cmap[ord(c)] for c in text]
    bp = BoundsPen(gs); x = 0; pos = []
    for n in names:
        pos.append(x); gs[n].draw(TransformPen(bp, (1, 0, 0, 1, x, 0))); x += gs[n].width + track
    x0, y0, x1, y1 = bp.bounds
    def path(cx, top=None, bottom=None):
        ox = cx - (x0 + (x1 - x0) / 2) * s
        oy = (top + y1 * s) if top is not None else (bottom + y0 * s)
        sp = SVGPathPen(gs)
        for n, px in zip(names, pos):
            gs[n].draw(TransformPen(sp, (s, 0, 0, -s, ox + px * s, oy)))
        return sp.getCommands()
    return path, ((x1 - x0) * s, (y1 - y0) * s)

def svg(w, h, body, defs=""):
    return f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {w} {h}" role="img" aria-label="nex-ovia"><defs>{defs}</defs>{body}</svg>\n'

def half_clip(id_, cx, cy, rx, ry, top, gap):
    # half of an ellipse, cut at cy -/+ gap
    if top:
        y = cy - gap; dx = rx * math.sqrt(1 - (gap / ry) ** 2)
        d = f"M{cx - dx},{y} A{rx},{ry} 0 0 1 {cx + dx},{y} Z"
    else:
        y = cy + gap; dx = rx * math.sqrt(1 - (gap / ry) ** 2)
        d = f"M{cx + dx},{y} A{rx},{ry} 0 0 1 {cx - dx},{y} Z"
    return f'<clipPath id="{id_}"><path d="{d}"/></clipPath>', d

# ---- A: the name is the O. Heavy wide letters fill each half and are cropped by the oval's edge.
def dir_a(ink_top, ink_bot, w=900, size=232, gap=9, rx=200, ry=150, track=-20):
    W, H = 440, 340; cx, cy = W / 2, H / 2
    ct, _ = half_clip("at", cx, cy, rx, ry, True, gap)
    cb, _ = half_clip("ab", cx, cy, rx, ry, False, gap)
    nex, _ = outline("nex", w, size, track)
    via, _ = outline("via", w, size, track)
    body = (f'<path d="{nex(cx, bottom=cy - gap - 2)}" fill="{ink_top}" clip-path="url(#at)"/>'
            f'<path d="{via(cx, top=cy + gap + 2)}" fill="{ink_bot}" clip-path="url(#ab)"/>')
    return svg(W, H, body, ct + cb)

# ---- B: hairline oval, light wide-tracked lowercase.
def dir_b(ink, accent):
    W, H = 440, 300; cx, cy = W / 2, H / 2; rx, ry, gap = 200, 128, 7
    _, dt = half_clip("bt", cx, cy, rx, ry, True, gap)
    _, db = half_clip("bb", cx, cy, rx, ry, False, gap)
    nex, _ = outline("nex", 300, 74, 160)
    via, _ = outline("via", 300, 74, 160)
    body = (f'<path d="{dt}" fill="none" stroke="{ink}" stroke-width="2.5"/>'
            f'<path d="{db}" fill="none" stroke="{accent}" stroke-width="2.5"/>'
            f'<path d="{nex(cx, bottom=cy - gap - 30)}" fill="{ink}"/>'
            f'<path d="{via(cx, top=cy + gap + 30)}" fill="{accent}"/>')
    return svg(W, H, body)

# ---- C: the step. Two solid halves, bottom shifted forward; letters knocked out.
def dir_c(top, bot, shift=26):
    W, H = 470, 300; cx, cy = 220, H / 2; rx, ry, gap = 196, 128, 6
    _, dt = half_clip("ct", cx, cy, rx, ry, True, gap)
    _, db = half_clip("cb", cx + shift, cy, rx, ry, False, gap)
    nex, _ = outline("nex", 800, 112, -6)
    via, _ = outline("via", 800, 112, -6)
    defs = (f'<mask id="mt"><rect width="{W}" height="{H}" fill="#fff"/><path d="{nex(cx, bottom=cy - gap - 22)}" fill="#000"/></mask>'
            f'<mask id="mb"><rect width="{W}" height="{H}" fill="#fff"/><path d="{via(cx + shift, top=cy + gap + 22)}" fill="#000"/></mask>')
    body = f'<path d="{dt}" fill="{top}" mask="url(#mt)"/><path d="{db}" fill="{bot}" mask="url(#mb)"/>'
    return svg(W, H, body, defs)

files = {
    "A-mono-dark.svg": dir_a("#f2f1ec", "#f2f1ec"),
    "A-duo-dark.svg": dir_a("#f2f1ec", "#00d4aa"),
    "A-mono-light.svg": dir_a("#14131c", "#14131c"),
    "B-dark.svg": dir_b("#f2f1ec", "#00d4aa"),
    "C-dark.svg": dir_c("#6c63ff", "#00d4aa"),
    "C-light.svg": dir_c("#14131c", "#14131c"),
}
if __name__ == "__main__":
  for n, s in files.items():
    open(f"{sys.argv[2]}/{n}", "w").write(s)
  print("wrote", len(files))
