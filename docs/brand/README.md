# nex-ovia mark: "the step"

The oval is the O of nex-o-via, parted by a horizontal cut. The top half (`nex`) steps back, the bottom half
(`via`) steps forward: next, by way of. Letters are Unbounded ExtraBold (SIL OFL), outlined and knocked out, so
the files need no font.

- `brand/nex-ovia-mark.svg`: indigo `#6c63ff` / teal `#00d4aa`, for dark backgrounds
- `brand/nex-ovia-mark-light.svg`: one-colour ink `#14131c`, for light backgrounds and print
- `brand/nex-ovia-mark-white.svg`: one-colour, on photos, video and brand colour
- `brand/nex-ovia-symbol*.svg`: letterless, for favicons and anything under ~32px
- `brand/avatar-1080.png`, `linkedin-banner-1584x396.png`, `x-header-1500x500.png`, `youtube-banner-2560x1440.png`

Rebuild: put Unbounded-800.ttf (instanced from https://github.com/google/fonts/tree/main/ofl/unbounded) in
`docs/brand/font/`, then `python mark.py <out-dir>` (needs `fonttools`).
