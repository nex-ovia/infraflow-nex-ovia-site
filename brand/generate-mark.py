"""nex-ovia mark: an oval O split by a horizontal cut. 'nex' stands on the cut (solid top half),
'via' hangs from it (open bottom bowl). Glyphs are outlined from DM Serif Display so no font is needed."""
import sys
from fontTools.ttLib import TTFont
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.pens.transformPen import TransformPen
from fontTools.pens.boundsPen import BoundsPen

font = TTFont(sys.argv[1]); out = sys.argv[2]
gs = font.getGlyphSet(); cmap = font.getBestCmap()

W, H = 400, 260            # viewBox
CX, CY = 200, 130          # oval centre
RX, RY = 184, 118          # oval radii
GAP = 7                    # half-gap of the cut
STROKE = 9                 # bowl stroke
TRACK = 18                 # letter-spacing in font units

def word(text, size, x_center, y_anchor, hang):
    """Outline `text` at `size` px; stand its bbox bottom on y_anchor, or hang its bbox top from it."""
    s = size / font['head'].unitsPerEm
    names = [cmap[ord(c)] for c in text]
    adv = [gs[n].width + TRACK for n in names]; total = sum(adv) - TRACK
    bp = BoundsPen(gs); x = 0
    for n, a in zip(names, adv):
        tp = TransformPen(bp, (1, 0, 0, 1, x, 0)); gs[n].draw(tp); x += a
    xmin, ymin, xmax, ymax = bp.bounds
    ox = x_center - (xmin + (xmax - xmin) / 2) * s
    oy = (y_anchor + ymax * s) if hang else (y_anchor + ymin * s)
    sp = SVGPathPen(gs); x = 0
    for n, a in zip(names, adv):
        tp = TransformPen(sp, (s, 0, 0, -s, ox + x * s, oy)); gs[n].draw(tp); x += a
    return sp.getCommands(), (ox + xmin * s, ox + xmax * s, oy - ymax * s, oy - ymin * s)

def inside(box, margin):
    x0, x1, y0, y1 = box
    return all(((x - CX) / (RX - margin)) ** 2 + ((y - CY) / (RY - margin)) ** 2 <= 1 for x in (x0, x1) for y in (y0, y1))

def fit(text, anchor, hang, margin):
    size = 160
    while size > 40:
        d, box = word(text, size, CX, anchor, hang)
        if inside(box, margin): return size
        size -= 1

SIZE = min(fit("nex", CY - GAP - 12, False, 16), fit("via", CY + GAP + 12, True, STROKE + 14))
nex = word("nex", SIZE, CX, CY - GAP - 12, False)[0]
via = word("via", SIZE, CX, CY + GAP + 12, True)[0]
print("size", SIZE)

def mark(top, bowl, ink, letters=True, stroke=STROKE):
    top_y, bot_y = CY - GAP, CY + GAP
    knock = f'<path d="{nex}" fill="#000"/>' if letters else ""
    via_p = f'<path d="{via}" fill="{ink}"/>' if letters else ""
    return f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {W} {H}" role="img" aria-label="nex-ovia">
  <defs>
    <clipPath id="t"><rect x="0" y="0" width="{W}" height="{top_y}"/></clipPath>
    <clipPath id="b"><rect x="0" y="{bot_y}" width="{W}" height="{H - bot_y}"/></clipPath>
    <mask id="m"><rect width="{W}" height="{H}" fill="#fff"/>{knock}</mask>
  </defs>
  <ellipse cx="{CX}" cy="{CY}" rx="{RX}" ry="{RY}" fill="{top}" clip-path="url(#t)" mask="url(#m)"/>
  <ellipse cx="{CX}" cy="{CY}" rx="{RX - stroke / 2}" ry="{RY - stroke / 2}" fill="none" stroke="{bowl}" stroke-width="{stroke}" clip-path="url(#b)"/>
  {via_p}
</svg>
'''

variants = {
    "nex-ovia-mark.svg":        ("#6c63ff", "#00d4aa", "#00d4aa", True),   # dark backgrounds
    "nex-ovia-mark-light.svg":  ("#4f46e5", "#00a383", "#00a383", True),   # light backgrounds
    "nex-ovia-mark-white.svg":  ("#ffffff", "#ffffff", "#ffffff", True),   # one-colour on photos/video
    "nex-ovia-mark-black.svg":  ("#0a0a0f", "#0a0a0f", "#0a0a0f", True),   # one-colour print
    "nex-ovia-symbol.svg":      ("#6c63ff", "#00d4aa", "#00d4aa", False),  # no letters: favicon / tiny sizes
}
for name, (t, b, i, l) in variants.items():
    open(f"{out}/{name}", "w").write(mark(t, b, i, l, STROKE if l else 30))
print("wrote", len(variants))
