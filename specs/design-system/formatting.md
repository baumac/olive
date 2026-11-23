# Dates, numbers and language

Rules for how the UI shows values from the API. Format every value with the browser's `Intl` APIs. Numbers and relative times use the `en` locale. Dates use `en-GB`, which puts the day first as in "4 Nov 2024". `en` gives "Nov 4, 2024". Never build date or number text by hand.

## Dates and times

The API sends UTC timestamps in the RFC 3339 format, `2024-11-04T01:26:55.002Z`. `new Date()` reads the value. See decision 00001.

- Within the last 7 days, show relative time: "just now" (under 1 minute), "5 minutes ago", "3 hours ago", "yesterday", "4 days ago". Use `Intl.RelativeTimeFormat` with `numeric: "auto"`.
- Older than 7 days, show the day, the short month and the year: "4 Nov 2024". Use `Intl.DateTimeFormat` with `day: "numeric", month: "short", year: "numeric"`.
- Show the full date and time in the person's time zone on hover and to screen readers: wrap the text in `<time datetime="2024-11-04T01:26:55.002Z" title="4 November 2024, 02:26">`.
- When `updated` is later than `created`, add "edited" after the date: "4 Nov 2024 · edited". Do not show both dates.
- Metadata lines join parts with a middle dot and spaces: "Ana · 4 Nov 2024".
- The `date` of a review is a local date without a time. Show it as "Visited on 14 Nov 2026" or "Cooked on 22 Nov 2026", never as relative time. `parseDay` of `lib/format.ts` reads it. See decision 00005.
- The table groups its entries by `created`: "Past week" for the last 7 days, then the month, such as "Earlier in November" for the current month and "October 2026" for an older one. The journal groups its entries by the month of `date`. Use `Intl.DateTimeFormat` with `month: "long", year: "numeric"`.

## Numbers and scores

- A score is a number without its maximum: "8", never "8 / 10" and never dots. Use lining `tabular-nums`. See decision 00009.
- The page of a subject or a review shows "out of 10" one time, under the overall of the score block.
- Restaurant averages come from the API with one decimal: "7.6". Show a whole average without a decimal: "9".
- The `overall` of a recipe review has up to two decimals, such as 8.25. Show it with one decimal, like the averages: "8.3". See decision 00004.
- The total time of a recipe is in minutes. Show it with `Intl.NumberFormat` units and `unitDisplay: "short"`: "45 min", "1 hr", "1 hr 30 min". `formatMinutes` of `lib/format.ts` does this.
- Do not average or round scores in the UI. Show the values of the API. The one exception is the overall that the review form shows before the save. It uses the formula of the API. See decision 00005.
- Distances show with one decimal below 10 km, such as "1.2 km", and as whole kilometers from 10 km. A distance under 100 m is "Less than 100 m". See decision 00008.
- Counts use `Intl.NumberFormat` and the right plural: "1 review", "12 reviews", "1,204 reviews". Use `Intl.PluralRules` to pick the word.
- Zero is a word in empty states ("No reviews yet") and a number in counts ("0 reviews").

## Text from the API

- Show `name`, `title`, `description` and `feedback` as typed. Keep line breaks in `description` and `feedback` with `white-space: pre-line`. Do not render Markdown or HTML.
- Long text in a card stops after 3 lines with an ellipsis (`line-clamp-3`). The detail screen shows all of it.
- Tags show as stored: lowercase, with hyphens.
- A creator that is `null` shows as "Deleted user".

## Language

- The UI supports only English, with `<html lang="en">`. Other languages are out of scope (ADR 00009). Do not add a language switcher or translation files.
- Format numbers with the `en` locale and dates with `en-GB`, as in the examples above. The page language stays `en`.
