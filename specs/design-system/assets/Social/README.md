# Social

`og-image.png` (1200×630): the image that shows when someone shares a link to food.sponga47.com in social media or a chat app. It has the logo, the tagline "What everyone ate and cooked lately.", the line "A food journal for friends and family.", the domain and an entry with a score stamp, on `background`.

Use it as the default for every page. A restaurant or review page can add its own title and description, but keeps this image until per-page images exist.

```html
<meta property="og:site_name" content="Olive">
<meta property="og:type" content="website">
<meta property="og:title" content="Olive, a food journal">
<meta property="og:description" content="Olive is a food journal for friends and family. What everyone ate and cooked lately.">
<meta property="og:image" content="https://food.sponga47.com/og-image.png">
<meta property="og:image:width" content="1200">
<meta property="og:image:height" content="630">
<meta property="og:image:alt" content="Olive logo with the tagline What everyone ate and cooked lately, and the line A food journal for friends and family.">
<meta name="twitter:card" content="summary_large_image">
```

Keep text out of the outer 60px, because some apps crop the edges.
