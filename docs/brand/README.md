# nex-ovia logo: "loop + waypoint"

The o of nexovia is an open loop. Its stroke doesn't stop: it leaves the loop, writes the v, and lands on the dot of
the i, which is teal. One line from an open question to a delivered result.

- Wordmark: `nex` and `a` are Bricolage Grotesque ExtraBold (SIL OFL), outlined; the o-v-i ligature is drawn to the
  font's own x-height and stem weight, so it reads as part of the word.
- Symbol: the ligature alone (loop → v → dot). Use it where the full word won't fit: avatars, favicons, app icons.
- Colours: ink `#0d0d0d`, paper `#f5f4f0`, teal `#00d4aa` (on light backgrounds `#00a383`). The teal is only ever the
  destination dot, or the symbol's stroke.

| File | Use |
|---|---|
| `brand/nex-ovia-wordmark.svg` | dark backgrounds (site header) |
| `brand/nex-ovia-wordmark-light.svg` | light backgrounds |
| `brand/nex-ovia-wordmark-white.svg` / `-black.svg` | one colour: photos, video, print |
| `brand/nex-ovia-symbol.svg` / `-light.svg` | symbol on dark / light, transparent |
| `brand/nex-ovia-app-icon.svg` | favicon and app icon tile |
| `brand/avatar-1080.png` | profile image for LinkedIn, X, Instagram, YouTube |
| `brand/linkedin-banner-1584x396.png`, `x-header-1500x500.png`, `youtube-banner-2560x1440.png` | profile banners |
| `brand/og-default-1200x630.png` | link preview image |

Rebuild: instance Bricolage Grotesque at wght 800 / opsz 96 / wdth 100 (from
https://github.com/google/fonts/tree/main/ofl/bricolagegrotesque), then `python mark.py <font.ttf> <out-dir>`
(needs `fonttools`).
