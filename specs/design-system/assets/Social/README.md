# Social

`og-image.png` (1200×630): the image that shows when someone shares a link to food.sponga47.com in social media or a chat app. It has the logo, the tagline "What everyone ate and cooked lately.", the domain and an entry with a score stamp, on `background`.

Use it as the default for every page. A restaurant or review page can add its own title and description, but keeps this image until per-page images exist.

```html
<meta property="og:site_name" content="Sponga47 Food">
<meta property="og:type" content="website">
<meta property="og:title" content="Sponga47 Food">
<meta property="og:description" content="What everyone ate and cooked lately.">
<meta property="og:image" content="https://food.sponga47.com/og-image.png">
<meta property="og:image:width" content="1200">
<meta property="og:image:height" content="630">
<meta property="og:image:alt" content="Sponga47 Food logo with the tagline What everyone ate and cooked lately.">
<meta name="twitter:card" content="summary_large_image">
```

Keep text out of the outer 60px, because some apps crop the edges.
