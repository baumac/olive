# Logos

The Olive logo: an olive with a leaf, and the name beside it. The olive is the name, and it ties the logo to the olive green palette and to food. "Olive" is set in TeX Gyre Bonum Bold at 40px with the kerning of the font, a serif, converted to outlines. No font file is needed. The UI sets names, headings and scores in Fraunces, a serif that sits well next to the logo, and all other text in the system sans.

- `olive-lockup-light.svg`: the full logo on light grounds (`background`, `card`). The olive in `primary` #4b6113, leaf and stem in `accent-foreground` #3d4f0e, "Olive" in `foreground` #1d1f14.
- `olive-lockup-dark.svg`: the full logo on dark grounds. The olive in dark `primary` #b7c878, leaf and stem in dark `accent-foreground` #cfdd95, "Olive" in dark `foreground` #f2ede0.
- `olive-mark-light.svg`, `olive-mark-dark.svg`: the olive alone, square, in the colors of the lockup of its theme. Use it for the favicon, app icons and small spaces where the name already shows. It stays legible at 16px.

Rules:

- Use the light files on light grounds and the dark files on dark grounds. Pick the file by theme. The colors are fixed in the SVG, so CSS cannot recolor them.
- Put the full logo in the page header, at 32 to 48px high. Below 24px high, use the mark alone.
- Keep clear space around the logo equal to half the olive's height.
- Give the image `alt="Olive"`. In a link to the home page, the alt text is the link name.
- Do not stretch, rotate, outline or recolor the logo. Do not set the name in another font next to the mark.
