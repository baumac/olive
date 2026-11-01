---
description: Store the Sponga47 design system in this repo and use it in the Vue Public UI.
status: proposed
---

# Motivation
[motivation]: #motivation

Sponga47 is the shared look for all self-hosted projects, including this one.
It defines tokens (color, type, spacing, radius), a brand book, and eight core components.
It is stored in `specs/design-system/`, next to the Public API specification.
Like the API spec, it is a contract that the `ui` app implements.

The reference components are React.
The Public UI is Vue.
We need a way for the Public UI to look and behave like Sponga47 without React,
and without the app drifting into its own version of the look.

Expected outcome:
- The Public UI uses the Sponga47 tokens and component styles directly from `specs/design-system/`.
- The Public UI has Vue versions of the eight core components with the same props, states, and accessibility.
- A token change is one edit to `specs/design-system/tokens.json` and one command.
- Light and dark themes both work, and follow the operating system by default.

# Scope
[scope]: #scope

In scope:
- The layout of `specs/design-system/`
- How the `ui` app loads the Sponga47 tokens and stylesheet
- The Vue component set and its API: `SpButton`, `SpTextField`, `SpSelect`, `SpCheckbox`, `SpCard`, `SpTabs`, `SpBadge`, `SpAlert`
- Theme switching (light, dark, follow the system)
- Font loading
- Tests for the Vue components

Out of scope:
- Scaffolding the `ui` app itself (Vite, router, API client). It is part of the "Design the UI" backlog item.
- Screen designs for the Public UI
- Moving Sponga47 into its own repository and package. See [Unresolved questions](#unresolved-questions).
- Components that Sponga47 does not define yet, such as a dialog, menu, or rating input.
  Add them to `specs/design-system/` first, then to the Vue set.
- A logo and an icon set. Sponga47 has neither yet.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

New concepts:
- **Sponga47**: the design system shared by all self-hosted projects.
  `specs/design-system/` is its source of truth for tokens, rules, and component behavior.
  Start with `specs/design-system/README.md`.
- **Token**: a named design value, such as the `brand` color or the `space-4` spacing step.
  In CSS, each token is a custom property, for example `var(--brand)`.
- **Sponga47 stylesheet**: `specs/design-system/components/bundle.css`.
  It is framework-agnostic, and every class starts with `sp-`.
  The React reference components and the Vue components render the same classes.

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

Changing a token:
1. Edit the value and usage note in `specs/design-system/tokens.json`.
2. Run `make design-tokens` to make `specs/design-system/tokens.css` again.
3. Commit both files. The `ui` app picks up the change on its next build.

# Implementation Details
[implementation-details]: #implementation-details

## Design system layout

```text
specs/design-system/
  README.md            brand book and file guide
  tokens.json          source of every token
  tokens.css           CSS form of tokens.json, committed
  fonts.css            optional Google Fonts import
  components/
    bundle.css         sp- component styles
    bundle.js          React 18 reference components
    index.d.ts         component props
    <Name>/README.md   usage guidelines
    <Name>/preview.html reference example
    Cover/preview.html  brand cover art
```

`tokens.css` is committed so that apps and previews can use it without a build step.
Until `make design-tokens` exists, `tokens.css` is edited by hand in the same change as `tokens.json`.
After that, a test fails when `tokens.css` does not match `tokens.json`.

## UI layout

```text
ui/src/sponga47/
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

The `ui` app imports the stylesheets straight from `specs/design-system/`. It keeps no copy.
A Vite alias `@design-system` points at `../specs/design-system`.
`server.fs.allow` includes that folder so the dev server can serve it.
`ui/src/main.ts` imports `@design-system/tokens.css` first, then `@design-system/components/bundle.css`, then app styles.

## Tokens

`make design-tokens` changes `tokens.json` into `tokens.css` in this layout:

```css
:root, [data-theme="light"] { --surface: #f4f6f8; --brand: #1d5fd1; /* ... */ }
[data-theme="dark"] { --surface: #0f1318; --brand: #5b9cf5; /* ... */ }
:root { --space-1: 4px; --radius-md: 8px; --font-sans: "Instrument Sans", system-ui, sans-serif; /* ... */ }
.sp-text-body { font-family: var(--font-sans); font-size: 16px; line-height: 24px; /* ... */ }
```

Type style classes use the `sp-text-` prefix, so they do not clash with app classes such as `.label`.

## Components

Each Vue component renders the same element structure and `sp-` classes as its React reference.
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

Behavior that must match the reference:
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

Font loading is separate from the component styles, in `specs/design-system/fonts.css`.
Self-hosted apps should not depend on a third-party font service.
They should also work on a network with no internet access.
So the `ui` app does not import `fonts.css`.
It installs the three fonts from `@fontsource` packages and imports them in `main.ts`.

## Tests

- A test checks that `tokens.css` matches `tokens.json`.
- Vitest and Vue Test Utils cover each component.
- Each component has tests for its variants, `v-model`, and ARIA attributes.
- SpTabs has keyboard tests for ArrowLeft, ArrowRight, Home, and End.
- `useTheme()` has tests for the saved choice and for following the system setting.

## Order of work

1. Add `make design-tokens` and the `tokens.css` check.
2. Scaffold `ui` with Vite, Vue 3.5 or later, and TypeScript. This belongs to the "Design the UI" backlog item.
3. Add the `@design-system` alias, `theme.ts`, the inline theme script, and the font packages.
4. Write the eight components and their tests.
5. Add a dev-only page that shows every component in both themes. Compare it with the reference previews.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

- Other self-hosted projects need Sponga47 too.
  Until it has its own repository, do they copy `specs/design-system/` from this repo?
- When Sponga47 moves to its own repository, should apps install it as a git dependency or from a private registry?
- Should the React reference components stay, or should the Vue components become the reference?
