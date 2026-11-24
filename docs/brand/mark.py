"""nex-ovia mark, "the step": the oval O of nex-o-via parted by a horizontal cut. The top half ('nex')
steps back, the bottom half ('via') steps forward. Letters are Unbounded ExtraBold, outlined and knocked out."""
import sys, math
sys.path.insert(0, sys.argv[0].rsplit('/', 1)[0])
from outline import outline, svg

W, H = 440, 300; CX, CY = 220, 150
RX, RY = 196, 128; GAP = 7; STEP = 16; MARGIN = 10

def half(cx, top):
    y = CY - GAP if top else CY + GAP
    dx = RX * math.sqrt(1 - (GAP / RY) ** 2)
    return (f"M{cx - dx:.2f},{y} A{RX},{RY} 0 0 1 {cx + dx:.2f},{y} Z" if top
            else f"M{cx + dx:.2f},{y} A{RX},{RY} 0 0 1 {cx - dx:.2f},{y} Z")

def inside(cx, x0, x1, y0, y1, top):
    for x in (x0, x1):
        for y in (y0, y1):
            if ((x - cx) / (RX - MARGIN)) ** 2 + ((y - CY) / (RY - MARGIN)) ** 2 > 1: return False
    return (y1 <= CY - GAP - MARGIN * .8) if top else (y0 >= CY + GAP + MARGIN * .8)

def place(text, size, cx, top):
    path, (w, h) = outline(text, 800, size, -4)
    # optical centre of each half sits ~40% of the way from the cut to the rim
    c = CY - GAP - 0.40 * (RY - GAP) if top else CY + GAP + 0.40 * (RY - GAP)
    y0, y1 = c - h / 2, c + h / 2
    return path(cx, top=y0), (cx - w / 2, cx + w / 2, y0, y1)

def fit():
    for size in range(160, 40, -1):
        ok = all(inside(cx, *place(t, size, cx, top)[1], top) for t, cx, top in (("nex", CX - STEP, True), ("via", CX + STEP, False)))
        if ok: return size

SIZE = fit()

def mark(top_fill, bot_fill, letters=True, bg_knock=True):
    nex, _ = place("nex", SIZE, CX - STEP, True); via, _ = place("via", SIZE, CX + STEP, False)
    if not letters:
        return svg(W, H, f'<path d="{half(CX - STEP, True)}" fill="{top_fill}"/><path d="{half(CX + STEP, False)}" fill="{bot_fill}"/>')
    defs = (f'<mask id="mt"><rect width="{W}" height="{H}" fill="#fff"/><path d="{nex}" fill="#000"/></mask>'
            f'<mask id="mb"><rect width="{W}" height="{H}" fill="#fff"/><path d="{via}" fill="#000"/></mask>')
    return svg(W, H, f'<path d="{half(CX - STEP, True)}" fill="{top_fill}" mask="url(#mt)"/><path d="{half(CX + STEP, False)}" fill="{bot_fill}" mask="url(#mb)"/>', defs)

if __name__ == "__main__":
    out = sys.argv[1]
    v = {
        "nex-ovia-mark.svg":        ("#6c63ff", "#00d4aa", True),
        "nex-ovia-mark-ink.svg":    ("#14131c", "#14131c", True),
        "nex-ovia-mark-white.svg":  ("#f4f3ee", "#f4f3ee", True),
        "nex-ovia-symbol.svg":      ("#6c63ff", "#00d4aa", False),
        "nex-ovia-symbol-ink.svg":  ("#14131c", "#14131c", False),
    }
    for n, (t, b, l) in v.items(): open(f"{out}/{n}", "w").write(mark(t, b, l))
    print("size", SIZE, "wrote", len(v))
