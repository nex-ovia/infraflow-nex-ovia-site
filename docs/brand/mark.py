"""nex-ovia mark: "loop + waypoint".
The o of nexovia is an open loop. Its stroke keeps going, writes the v, and lands on the dot of the i (the
destination, in teal). 'nex' and 'a' are Bricolage Grotesque ExtraBold (SIL OFL), outlined; the o-v-i ligature is
drawn to the font's measured x-height (52.8) and stem weight (17) at 100 units = 1 em, baseline y = 0.
Usage: python mark.py BricolageGrotesque-800.ttf <out-dir>"""
import sys
from fontTools.ttLib import TTFont
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.pens.transformPen import TransformPen
from fontTools.pens.boundsPen import BoundsPen

font = TTFont(sys.argv[1]); OUT = sys.argv[2]
gs = font.getGlyphSet(); cmap = font.getBestCmap()
S = 100 / font['head'].unitsPerEm          # 100 units = 1 em
TRACK = -4                                   # letter-spacing -0.04em

LIG = "M41.5,-10.8 A19.3,19.3 0 1 1 42.9,-43.3 L70,-8.5 L92,-44.3"   # open o that writes the v
LIG_W = 120                                  # ligature advance; i stem at x 103..119.5
STEM = (103, -52.8, 16.5, 52.8)
DOT = (111.2, -65, 9.5)

def text(t, x0):
    sp = SVGPathPen(gs); x = x0
    for c in t:
        g = gs[cmap[ord(c)]]
        g.draw(TransformPen(sp, (S, 0, 0, -S, x, 0))); x += g.width * S + TRACK
    return sp.getCommands(), x

nex, x = text("nex", 0)
lig_x = x                                    # ligature follows 'nex' (after its tracking)
a, end = text("a", lig_x + LIG_W)
bp = BoundsPen(gs); gs[cmap[ord("a")]].draw(bp)

TOP, BOT = -78, 4                            # dot top ~ -74.5; overshoot below baseline
L, R = -2, end - TRACK + 2

def wordmark(ink, dot):
    sx, sy, sw, sh = STEM
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="{L} {TOP} {R - L:.1f} {BOT - TOP}" role="img" aria-label="nexovia">'
            f'<path d="{nex}" fill="{ink}"/>'
            f'<g transform="translate({lig_x:.2f} 0)"><path d="{LIG}" fill="none" stroke="{ink}" stroke-width="17" stroke-linecap="round" stroke-linejoin="round"/>'
            f'<rect x="{sx}" y="{sy}" width="{sw}" height="{sh}" rx="1" fill="{ink}"/><circle cx="{DOT[0]}" cy="{DOT[1]}" r="{DOT[2]}" fill="{dot}"/></g>'
            f'<path d="{a}" fill="{ink}"/></svg>\n')

def symbol(ink, dot, bg=None):
    # the ligature alone: open loop -> v -> destination dot, square viewBox
    rect = f'<rect x="-7" y="-104" width="132" height="132" rx="29" fill="{bg}"/>' if bg else ""
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="-7 -104 132 132" role="img" aria-label="nexovia symbol">{rect}'
            f'<path d="{LIG}" fill="none" stroke="{ink}" stroke-width="19" stroke-linecap="round" stroke-linejoin="round"/>'
            f'<circle cx="104" cy="-66" r="11" fill="{dot}"/></svg>\n')

files = {
    "nex-ovia-wordmark.svg":       wordmark("#f5f4f0", "#00d4aa"),   # dark backgrounds
    "nex-ovia-wordmark-light.svg": wordmark("#0d0d0d", "#00a383"),   # light backgrounds
    "nex-ovia-wordmark-white.svg": wordmark("#ffffff", "#ffffff"),   # one colour on photos/video
    "nex-ovia-wordmark-black.svg": wordmark("#0d0d0d", "#0d0d0d"),   # one colour print
    "nex-ovia-symbol.svg":         symbol("#00d4aa", "#f5f4f0"),     # transparent, dark UIs
    "nex-ovia-symbol-light.svg":   symbol("#0d0d0d", "#00a383"),     # transparent, light UIs
    "nex-ovia-app-icon.svg":       symbol("#00d4aa", "#f5f4f0", "#0d0d0d"),  # favicon / app icon tile
}
for n, s in files.items():
    open(f"{OUT}/{n}", "w").write(s)
print("wordmark width", round(R - L, 1), "lig at", round(lig_x, 1))
