"""nex-ovia mark, "horizon & path". The oval O of nex-o-via is parted by a horizontal cut (the horizon).
'nex' is knocked out of the solid half that rises above it. Below, 'via' sits in a bowl drawn as one tapering
calligraphic stroke: heavy at the left, thinning as it climbs, then leaving the O and running out along the
horizon to a single point (the destination; also the middle dot of nex·ovia). Letters: DM Serif Display, outlined."""
import sys, math
from fontTools.ttLib import TTFont
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.pens.transformPen import TransformPen
from fontTools.pens.boundsPen import BoundsPen

font = TTFont(sys.argv[1]); OUT = sys.argv[2]
gs = font.getGlyphSet(); cmap = font.getBestCmap(); UPM = font['head'].unitsPerEm

CX, CY, RX, RY, GAP = 200, 140, 180, 116, 6
W0, W1 = 17.0, 2.6                     # stroke weight: heavy start, hairline end
TAIL = 92                              # how far the path runs past the O
H = 280

def word(text, size, cx, anchor, hang, track=10):
    s = size / UPM; names = [cmap[ord(c)] for c in text]
    bp = BoundsPen(gs); x = 0; pos = []
    for n in names:
        pos.append(x); gs[n].draw(TransformPen(bp, (1, 0, 0, 1, x, 0))); x += gs[n].width + track
    x0, y0, x1, y1 = bp.bounds
    ox = cx - (x0 + (x1 - x0) / 2) * s
    oy = (anchor + y1 * s) if hang else (anchor + y0 * s)
    sp = SVGPathPen(gs)
    for n, px in zip(names, pos):
        gs[n].draw(TransformPen(sp, (s, 0, 0, -s, ox + px * s, oy)))
    return sp.getCommands(), (ox + x0 * s, ox + x1 * s, oy - y1 * s, oy - y0 * s)

def inside(b, mx, my, rx, ry):
    return all(((x - CX) / (rx - mx)) ** 2 + ((y - CY) / (ry - my)) ** 2 <= 1 for x in b[:2] for y in b[2:])

def fit():
    for size in range(150, 40, -1):
        _, bn = word("nex", size, CX, CY - GAP - 14, False)
        _, bv = word("via", size, CX, CY + GAP + 14, True)
        if inside(bn, 18, 12, RX, RY) and inside(bv, W0 + 16, W0 + 12, RX, RY): return size

SIZE = fit()
NEX, _ = word("nex", SIZE, CX, CY - GAP - 14, False)
VIA, _ = word("via", SIZE, CX, CY + GAP + 14, True)

def top_half():
    y = CY - GAP; dx = RX * math.sqrt(1 - (GAP / RY) ** 2)
    return f"M{CX - dx:.2f},{y} A{RX},{RY} 0 0 1 {CX + dx:.2f},{y} Z"

def path_shape(tail=True):
    """Tapered bowl as a filled outline; optional exit along the horizon."""
    ts = math.pi - math.asin(GAP / RY)          # left end at the cut
    te = math.asin((GAP + 11) / RY) if tail else math.asin(GAP / RY)   # stop a bit early to turn out
    N = 160; outer = []; inner = []
    for i in range(N + 1):
        t = ts + (te - ts) * i / N; u = i / N
        w = W0 * (1 - u) ** 1.15 + W1 * (1 - (1 - u) ** 1.15)
        outer.append((CX + RX * math.cos(t), CY + RY * math.sin(t)))
        inner.append((CX + (RX - w) * math.cos(t), CY + (RY - w) * math.sin(t)))
    d = "M" + " L".join(f"{x:.2f},{y:.2f}" for x, y in outer)
    if tail:
        ex, ey = outer[-1]; ix, iy = inner[-1]
        hy = CY + GAP                                    # the horizon the path joins
        xend = CX + RX + TAIL
        # outer edge curves out to the horizon, runs to the end; inner edge returns parallel
        d += f" C{ex + 0.8:.2f},{ey - 6:.2f} {ex + 3:.2f},{hy:.2f} {ex + 11:.2f},{hy:.2f} L{xend:.2f},{hy:.2f}"
        d += f" L{xend:.2f},{hy + W1:.2f} L{ix + 11:.2f},{hy + W1:.2f}"
        d += f" C{ix + 3:.2f},{hy + W1:.2f} {ix + 0.8:.2f},{iy - 5:.2f} {ix:.2f},{iy:.2f}"
    d += " L" + " L".join(f"{x:.2f},{y:.2f}" for x, y in reversed(inner)) + " Z"
    return d, (CX + RX + TAIL, CY + GAP + W1 / 2)

def mark(top, path, ink, letters=True, tail=True, dot=True):
    d, (dx, dy) = path_shape(tail)
    W = (CX + RX + TAIL + 22) if tail else (CX + RX + 20)
    vb_x = CX - RX - 20
    knock = f'<path d="{NEX}" fill="#000"/>' if letters else ""
    via = f'<path d="{VIA}" fill="{ink}"/>' if letters else ""
    end = f'<circle cx="{dx + 7:.2f}" cy="{dy:.2f}" r="6" fill="{ink}"/>' if (tail and dot) else ""
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="{vb_x} 0 {W - vb_x} {H}" role="img" aria-label="nex-ovia">'
            f'<defs><mask id="k"><rect x="{vb_x}" width="{W - vb_x}" height="{H}" fill="#fff"/>{knock}</mask></defs>'
            f'<path d="{top_half()}" fill="{top}" mask="url(#k)"/><path d="{d}" fill="{path}"/>{via}{end}</svg>\n')

V = {
    "nex-ovia-mark.svg":       ("#6c63ff", "#00d4aa", "#00d4aa", True, True),    # dark backgrounds
    "nex-ovia-mark-light.svg": ("#14131c", "#14131c", "#14131c", True, True),    # light backgrounds, print
    "nex-ovia-mark-white.svg": ("#f4f3ee", "#f4f3ee", "#f4f3ee", True, True),    # photos, video, brand colour
    "nex-ovia-symbol.svg":     ("#6c63ff", "#00d4aa", "#00d4aa", False, True),   # favicon, < 40px
    "nex-ovia-symbol-ink.svg": ("#14131c", "#14131c", "#14131c", False, True),
}
for n, (t, p, i, l, tl) in V.items(): open(f"{OUT}/{n}", "w").write(mark(t, p, i, l, tl))
print("size", SIZE)
