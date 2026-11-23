# App icons

Favicons and home screen icons, made from the olive mark. Put the files in `app/ui/public/` and link them from `index.html`.

- `favicon.svg`: the browser tab icon. It switches to the dark-theme olive when the system is dark, through a `prefers-color-scheme` rule inside the SVG.
- `favicon-32.png`: a fallback for browsers without SVG favicons.
- `apple-touch-icon.png` (180×180): the iPhone and iPad home screen icon. The olive sits on a solid `accent` tile (#ebf2dc), because iOS fills transparent areas with black.
- `icon-192.png`, `icon-512.png`: the web app manifest icons, same tile.
- `icon-maskable-512.png`: the Android adaptive icon. The olive stays inside the central safe zone, so any mask shape keeps it whole.

```html
<link rel="icon" href="/favicon.svg" type="image/svg+xml">
<link rel="icon" href="/favicon-32.png" sizes="32x32" type="image/png">
<link rel="apple-touch-icon" href="/apple-touch-icon.png">
<link rel="manifest" href="/site.webmanifest">
<meta name="theme-color" content="#ffffff" media="(prefers-color-scheme: light)">
<meta name="theme-color" content="#0c0c09" media="(prefers-color-scheme: dark)">
```

`site.webmanifest`:

```json
{
  "name": "Sponga47 Food Reviews",
  "short_name": "Sponga47",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#ffffff",
  "theme_color": "#546e0d",
  "icons": [
    { "src": "/icon-192.png", "sizes": "192x192", "type": "image/png" },
    { "src": "/icon-512.png", "sizes": "512x512", "type": "image/png" },
    { "src": "/icon-maskable-512.png", "sizes": "512x512", "type": "image/png", "purpose": "maskable" }
  ]
}
```
