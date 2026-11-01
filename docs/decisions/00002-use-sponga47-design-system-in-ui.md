---
description: Use the Sponga47 design system in the Vue Public UI.
status: proposed
---

# Motivation
[motivation]: #motivation

Sponga47 is the shared look for all self-hosted projects, including this one.
Its source of truth is the Sponga47 design system artifact:
https://claude.ai/artifact/H6kBoW5YMTn6B9NuEDaXkX

The design system defines tokens (color, type, spacing, radius), a brand book, and eight core components.
Its components are React, because the design system page renders its previews with React.
The Public UI is Vue.
We need a way for the Public UI to look and behave like Sponga47 without React,
and without each app drifting into its own version of the look.

Expected outcome:
- The Public UI uses the Sponga47 tokens and component styles unchanged.
- The Public UI has Vue versions of the eight core components with the same props, states, and accessibility.
- A token change in the design system reaches the Public UI by re-running one command.
- Light and dark themes both work, and follow the operating system by default.

# Scope
[scope]: #scope

In scope:
- How the `ui` app gets the Sponga47 tokens and stylesheet
- The Vue component set and its API: `SpButton`, `SpTextField`, `SpSelect`, `SpCheckbox`, `SpCard`, `SpTabs`, `SpBadge`, `SpAlert`
- Theme switching (light, dark, follow the system)
- Font loading
- Tests for the Vue components

Out of scope:
- Scaffolding the `ui` app itself (Vite, router, API client). It is part of the "Design the UI" backlog item.
- Screen designs for the Public UI
- Extracting Sponga47 into its own repository and package. See [Unresolved questions](#unresolved-questions).
- Components that the design system does not define yet, such as a dialog, menu, or rating input.
  Add them to the design system first, then to the Vue set.
- A logo and an icon set. The design system has neither yet.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

New concepts:
- **Sponga47**: the design system shared by all self-hosted projects.
  The design system artifact is the source of truth for tokens, rules, and component behavior.
- **Token**: a named design value, such as the `brand` color or the `space-4` spacing step.
  In CSS, each token is a custom property, for example `var(--brand)`.
- **Sponga47 stylesheet**: the framework-agnostic CSS for the components.
  Every class starts with `sp-`. The React and Vue components render the same classes.
- **Vendored copy**: the Sponga47 files that the `ui` app copies into `ui/src/sponga47/`.
  Nobody edits them in this repo. Changes go to the design system, then get synced.

Using a component in a Vue page:

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { SpButton, SpCard, SpTextField } from '@/sponga47'

const title = ref('')
const error = ref('')
</script>

<template>
  <SpCard title="New review">
    <SpTextField v-model="title" label="Title" :error="error" />
    <template #footer>
      <SpButton variant="ghost">Cancel</SpButton>
      <SpButton variant="primary" type="submit">Add review</SpButton>
    </template>
  </SpCard>
</template>
```

Using tokens in an app's own styles:

```css
.review-list {
  display: grid;
  gap: var(--space-4);
}
.review-meta {
  color: var(--ink-muted);
}
```

Apps never write raw hex values. They use tokens, so both themes work.

Syncing after the design system changes:

```bash
make sync-sponga47
```

This copies `tokens.json` and the stylesheet from the design system into `ui/src/sponga47/`.
It then makes `tokens.css` again. The diff shows exactly what changed.

# Implementation Details
[implementation-details]: #implementation-details

## File layout

```text
ui/src/sponga47/
  tokens.json        copied from the design system
  tokens.css         made from tokens.json by the sync script
  sponga47.css       copied from the design system's components/bundle.css
  theme.ts           theme state and the system-preference listener
  index.ts           exports the components and theme helpers
  components/
    SpButton.vue
    SpTextField.vue
    SpSelect.vue
    SpCheckbox.vue
    SpCard.vue
    SpTabs.vue
    SpBadge.vue
    SpAlert.vue
```

`ui/src/main.ts` imports `tokens.css` first, then `sponga47.css`, then app styles.

## Why vendor instead of a shared package

Only one app uses Sponga47 today.
A shared package needs its own repository, versioning, and a registry or git dependency.
That cost pays off only once a second app uses Sponga47.
Vendoring keeps the files in one folder, so moving them to a package later is a copy and an import change.

## Tokens

The design system stores tokens as `tokens.json`.
The sync script changes it into `tokens.css` in the same layout the design system page uses:

```css
:root, [data-theme="light"] { --surface: #f4f6f8; --brand: #1d5fd1; /* ... */ }
[data-theme="dark"] { --surface: #0f1318; --brand: #5b9cf5; /* ... */ }
:root { --space-1: 4px; --radius-md: 8px; --font-sans: "Instrument Sans", system-ui, sans-serif; /* ... */ }
.body { font-size: 16px; line-height: 24px; /* ... */ }
```

Making `tokens.css` from `tokens.json` in this repo keeps the token list as the only source.
It also removes the need for the design system page to save a `tokens.css` first.

## Components

Each Vue component renders the same element structure and `sp-` classes as its React version.
The shared stylesheet then gives both the same look.
Component names use the `Sp` prefix.
This avoids clashes with HTML elements and follows the Vue multi-word component name rule.

| React prop | Vue equivalent |
| --- | --- |
| `children` | default slot |
| `value` / `onChange` on TextField, Select, Checkbox | `v-model` (`modelValue`, `update:modelValue`) |
| `value` / `defaultValue` / `onChange` on Tabs | `v-model` on the selected tab id, defaulting to the first tab |
| Card `title`, `actions`, `footer` | `title` prop or `#title` slot, `#actions` slot, `#footer` slot |
| Tabs `tabs[].content` | a `#panel` scoped slot that receives the selected tab id |
| `label`, `hint`, `error` | props of the same name |
| `variant`, `size`, `tone` | props of the same name and values |
| `React.useId()` | Vue `useId()`, available from Vue 3.5 |

Behavior that must match the design system:
- Every interactive element shows a solid 2px `accent` focus ring on `:focus-visible`.
- TextField and Select connect hint and error text with `aria-describedby`. They set `aria-invalid` when `error` is set.
- Tabs use `role="tablist"`, `role="tab"`, and `role="tabpanel"`. Arrow keys, Home, and End move the selection. Only the selected tab is in the tab order.
- Alert uses `role="alert"` for `danger` and `role="status"` for other tones.
- Button defaults to `type="button"`.

## Theme

`tokens.css` uses light colors on `:root`. Dark colors need `data-theme="dark"` on `<html>`.
`theme.ts` exposes `useTheme()` with three choices: `light`, `dark`, and `system`.
The default is `system`.
`system` follows `prefers-color-scheme` and updates when the operating system setting changes.
The choice is saved in `localStorage` under `sponga47-theme`.

A short inline script in `ui/index.html` sets `data-theme` before the app mounts.
This prevents a flash of the light theme when the page loads in dark mode.

## Fonts

The design system stylesheet loads Bricolage Grotesque, Instrument Sans, and JetBrains Mono from Google Fonts.
Self-hosted apps should not depend on a third-party font service.
They should also work on a network with no internet access.
The sync script removes the Google Fonts `@import` from `sponga47.css`.
The `ui` app installs the three fonts from `@fontsource` packages and imports them in `main.ts`.

## Tests

- Vitest and Vue Test Utils cover each component.
- Each component has tests for its variants, `v-model`, and ARIA attributes.
- SpTabs has keyboard tests for ArrowLeft, ArrowRight, Home, and End.
- `useTheme()` has tests for the saved choice and for following the system setting.

## Order of work

1. Scaffold `ui` with Vite, Vue 3.5 or later, and TypeScript. This belongs to the "Design the UI" backlog item.
2. Add the sync script and the `make sync-sponga47` target. Commit the first vendored copy.
3. Add `theme.ts`, the inline theme script, and the font packages.
4. Write the eight components and their tests.
5. Add a dev-only page that shows every component in both themes. Compare it with the design system previews.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

- When a second app uses Sponga47, should it move to its own repository as a package (for example `@sponga47/vue`)?
  If yes, should apps install it as a git dependency or from a private registry?
- Should the design system keep the React components as the reference, or should it document the Vue API too?
  Its previews need React, so React stays there in either case.
- How does the sync script get files from the design system artifact?
  Options: download them by hand from the page's Files view, or use a script with an access token.
- Should the design system stylesheet drop the Google Fonts `@import` at the source, so no app needs to remove it?
