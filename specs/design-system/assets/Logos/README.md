# Logos

The Sponga47 Food Reviews logo: an olive with a leaf, and the name beside it. The olive ties the logo to the olive green palette and to food. "Sponga47" is set in TeX Gyre Bonum Bold and "Food Reviews" in TeX Gyre Bonum Regular, a serif, both converted to outlines. The lockup files still draw "Reviews" without "Food". The backlog has the change. No font file is needed. The serif appears only in the logo; the UI keeps the system sans.

- `sponga47-reviews-lockup-light.svg`: the full logo on light grounds (`background`, `card`). Olive and "Food Reviews" in `primary` #546e0d, leaf and stem in `accent-foreground` #41560a, "Sponga47" in olive-950 #0c0c09.
- `sponga47-reviews-lockup-dark.svg`: the full logo on dark grounds. Olive and "Food Reviews" in dark `primary` #a4c665, leaf and stem in dark `accent-foreground` #d7e5b9, "Sponga47" in olive-50 #fbfbf9.
- `sponga47-reviews-mark-light.svg`, `sponga47-reviews-mark-dark.svg`: the olive alone, square. Use it for the favicon, app icons and small spaces where the name already shows. It stays legible at 16px.

Rules:

- Use the light files on light grounds and the dark files on dark grounds. Pick the file by theme. The colors are fixed in the SVG, so CSS cannot recolor them.
- Put the full logo in the page header, at 32 to 48px high. Below 24px high, use the mark alone.
- Keep clear space around the logo equal to half the olive's height.
- Give the image `alt="Sponga47 Food Reviews"`. In a link to the home page, the alt text is the link name.
- Do not stretch, rotate, outline or recolor the logo. Do not set the name in another font next to the mark.
