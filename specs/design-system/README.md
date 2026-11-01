# Sponga47

Sponga47 is the shared look for a family of self-hosted web apps. Each app keeps its own name and content; Sponga47 keeps them feeling like one toolbox: cool, calm, and practical. Use these rules in every app.

## Files

This folder is the source of truth for Sponga47.

| Path | What |
| --- | --- |
| `tokens.json` | Every token, with its value per theme and a usage note. Edit tokens here. |
| `tokens.css` | The tokens as CSS custom properties, plus one `sp-text-<style>` class per type style. It has no generator yet, so update it in the same change as `tokens.json`. |
| `fonts.css` | Loads the three font families from Google Fonts. Optional: apps can self-host the fonts instead. |
| `components/bundle.css` | The framework-agnostic component styles. Every class starts with `sp-`. |
| `components/bundle.js` | The React 18 reference components, as one classic script that sets `window.Sponga47`. |
| `components/index.d.ts` | The component props, as TypeScript types. |
| `components/<Name>/README.md` | Usage guidelines for one component. |
| `components/<Name>/preview.html` | A reference example of one component. Open it in a browser; it loads the stylesheets and `bundle.js` from this folder and React 18.3.1 from jsDelivr. |

Load the stylesheets in this order: `tokens.css`, then `components/bundle.css`, then app styles.

## Content fundamentals

- Write in plain, direct sentences. Address the reader as "you"; the app never says "we".
- Use sentence case for titles, buttons and labels ("Add review", not "Add Review").
- Name actions with a verb: "Save", "Delete review", "Sign in". Avoid "OK" and "Submit".
- No emoji in UI copy. No exclamation marks in errors.
- Errors say what happened and what to do next: "The server did not respond. Try again in a minute."

## Color

- Every app ships Light and Dark. Follow the operating system setting by default and let the user override it. Use only tokens so both themes work; never hard-code a color.
- The palette is cool: slate greys for structure, blue for action. Keep warm hues for `danger` only.
- Set every page on `surface`. Put cards, panels, inputs and menus on `surface-raised`.
- Set text in `ink`. Use `ink-muted` for secondary text only, never for body copy.
- `brand` blue is the one loud color. Use it for the primary action on a screen and for highlights, always with `on-brand` text. Use it as text only for short emphasis, never for body copy.
- `accent` cyan marks links, focus and selection. Links are `accent` and underlined. Text on an `accent`, `success` or `danger` fill is `on-accent`, never literal white.
- `success` and `danger` are for state only. Always pair them with a word or an icon; never rely on color alone.
- Separate regions with `line`. Give inputs and other controls a `line-strong` border.

## Type

- Headings use the display family (`display`, `heading`, `subheading`). Use `display` at most once per page.
- Running text is `body`. Use `small` for metadata and table cells, `label` for form labels and badges.
- Use `code` for IDs, API tokens and log output.
- Apply a type style with its class, for example `sp-text-body` or `sp-text-label`.
- Self-host Bricolage Grotesque, Instrument Sans and JetBrains Mono with the app, for example from `@fontsource` packages, so the app works offline. Use `fonts.css` (Google Fonts) only for quick prototypes.

## Spacing, radius and layout

- Build every gap from `space-1`, `space-2`, `space-4` and `space-8`.
- Pad cards with `space-4`. Separate page sections with `space-8`. Use a `space-4` gutter on phones.
- Round buttons, cards and panels with `radius-md`; inputs and badges with `radius-sm`; pills and avatars with `radius-full`.
- Prefer borders to shadows. Keep layouts single-column on phones with no horizontal scroll.

## States and focus

- The focus ring is a solid 2px `accent` outline with a 2px offset, on every interactive element.
- Disabled controls keep their layout and drop to `ink-muted` text with a `line` border.

## Components

- Build screens from the Sponga47 components first: Button, TextField, Select, Checkbox, Card, Tabs, Badge and Alert. They need React 18 and the token stylesheet.
- Their classes start with `sp-`. Don't restyle them per app; change a token instead so every app moves together.

## Logo and iconography

- Sponga47 has no logo mark yet. Set the name in the `heading` style in `ink`, with the `47` unchanged.
- No icon set is chosen yet. Until one is, use simple 1.5px-stroke outline icons in `ink`, sized 16px or 20px.
