---
description: Add a design system to the Vue Public UI.
status: implemented
---

# Motivation
[motivation]: #motivation

The Public UI does not exist yet.
Before we build screens, we need one shared set of design tokens and components.
Without it, each screen makes its own choices for color, spacing, and form controls.
The UI then looks inconsistent, and accessibility problems repeat in many places.

Expected outcome:
- All screens use the same tokens for color, type, spacing, and radius.
- All screens use the same accessible components for forms, feedback, and layout.
- The components cover the needs of the Public API, such as the five review scores.
- Light and dark themes both work. The default follows the operating system.
- The Public UI has one brand: a name, a logo, a palette, and app icons.

# Scope
[scope]: #scope

In scope:
- The choice of design system for the UI in `app/ui`
- The design tokens and how the UI in `app/ui` defines them
- The first component set and where it lives in the UI in `app/ui`
- Theme switching
- The brand: the name, the palette, the logo, the app icons, and the share image
- Rules for dates, numbers, and text from the API
- Accessibility rules for components
- Tests for the components

Out of scope:
- Scaffolding the UI in `app/ui` itself, such as Vite, the router, and the API client. Decision 00005 covers it.
- Screen designs for the Public UI
- An icon set of our own. The UI uses Lucide.
- Languages other than English. The UI supports only English.
- A visual catalog of the components, such as Storybook or Histoire
- Sharing the design system with other projects

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

New concepts:
- **Design system**: the tokens, components, and rules that all screens of the Public UI use.
- **Token**: a named design value, such as the `primary` color or the `radius` value.
  In CSS, each token is a custom property, for example `var(--primary)`.
- **Component**: a Vue component in `app/ui/src/components/ui/`, such as `Button` or `RatingInput`.
  Screens use these components. They do not style native form controls directly.

The design system uses [shadcn-vue](https://www.shadcn-vue.com/) with [Tailwind CSS](https://tailwindcss.com/).
shadcn-vue is not a package dependency.
Its CLI copies the source of each component into `app/ui/src/components/ui/`.
The project owns this code and can change it.
The components use [Reka UI](https://reka-ui.com/) for accessible behavior, such as focus, keyboard control, and ARIA attributes.

Using a component in a Vue page:

```vue
<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { RatingInput } from '@/components/ui/rating-input'

const food = defineModel<number>({ required: true })
</script>

<template>
  <RatingInput v-model="food" :max="10" label="Food" />
  <Button type="submit">Save review</Button>
</template>
```

## First component set

The first set contains the components that the Public API needs.

From shadcn-vue:
- `Button`
- `Input` and `Textarea`, for `name`, `title`, `description`, and `feedback`
- `Label` and `Form`, for labels, help text, and validation messages
- `Select`, for sort order and filters
- `Card`, for restaurants and reviews in lists
- `Badge`, for tags
- `Alert`, for errors from the API
- `Dialog`, to confirm deletes
- `Skeleton`, for loading states
- `Avatar`, for the creator of a restaurant or a review
- `Sonner`, for toast messages after create, edit, and delete
- `DropdownMenu`, for the theme menu and the account menu

Project components that shadcn-vue does not have:
- `RatingInput`: an integer score from 1 to a maximum. Reviews use five of them with a maximum of 10.
- `RatingDisplay`: a read-only score from 1 to a maximum
- `TagInput`: adds and removes tags. It checks the [Tags](00001-define-public-api-spec.md#tags) rules before the API does.
- `LoadMore`: gets the next page of a list with the `next_cursor` value. See [Pagination](00001-define-public-api-spec.md#pagination).
- `ThemeToggle`: selects light, dark, or system.
- `UserMenu`: shows the signed in user and the sign out action.
- `SignInForm`: gets the email and the password.
- `FilterBar`: sets the filters and the sort order of a list. It shows each active filter as a chip.
- `EmptyState`: fills the place of a list with no items.

App components that know the routes. They live in `app/ui/src/components/app/`:
- `AppHeader`: the logo, the main navigation, `ThemeToggle`, and `UserMenu`
- `ErrorPage`: the full page for `404`, `403`, and `500` errors

A new component goes into the set only when a screen needs it.

## Themes

The UI has a light theme and a dark theme.
The default follows the `prefers-color-scheme` setting of the operating system.
A user can select light, dark, or system.
The UI stores this selection in `localStorage`.

# Implementation Details
[implementation-details]: #implementation-details

## Why shadcn-vue

We compared these options:

| Option | Result |
| --- | --- |
| shadcn-vue with Tailwind CSS | Selected. We own the component code. Reka UI gives accessible behavior. Tokens are CSS custom properties. |
| Vuetify | Not selected. It uses Material Design. Changing the look away from Material takes a lot of work. |
| PrimeVue | Not selected. It is a large dependency. Its styled mode is hard to change. Its unstyled mode needs a full set of our own styles. |
| A custom design system | Not selected. Accessible selects, tabs, and dialogs take a lot of work to build and test. |

## Tokens

- tokens are CSS custom properties in `app/ui/src/assets/main.css`
- the light theme defines tokens on `:root`. The dark theme defines the same tokens on `.dark`.
- Tailwind CSS uses the tokens through its `@theme` configuration
- components use Tailwind classes that refer to tokens, such as `bg-primary`. They do not use fixed color values.
- the tokens use the shadcn-vue names: `background`, `foreground`, `card`, `popover`, `primary`, `secondary`, `muted`, `accent`, `destructive`, `border`, `input`, `ring`, and `radius`
- the neutral colors use the olive base color. Its values come from the Tailwind CSS v4.2 olive scale.
- `primary`, `ring`, and `accent` use the olive green brand color
- `success` is blue. It marks confirmations. Blue does not look like the green brand color or like the red `destructive` color.
- these values are not the shadcn-vue values. They meet the contrast rules in both themes:
  - `muted-foreground` is darker than olive-500 in the light theme
  - `input` is darker in the light theme and white at 16% in the dark theme
  - `ring` is a solid 2px outline. It does not use 50% opacity.
  - the dark `destructive` fill is 50% over `background`, not 60%
- the layout tokens are the Tailwind CSS v4 breakpoints, the `content-max` and `reading-max` widths, and the `z-header`, `z-popover`, and `z-toast` layers

## Brand

- the product name is "Sponga47 Reviews". The UI is at reviews.sponga47.com.
- the logo is an olive with a leaf, next to the name. The name uses the TeX Gyre Bonum serif as outlines. The UI text uses the system sans font.
- each logo file has a light version and a dark version. The colors are fixed in the SVG.
- the favicon, the home screen icons, and the share image come from the logo
- the Sponga47 Reviews design system artifact keeps the tokens, the component rules, the logo files, and the icons

## Theme switching

- the UI adds or removes the `dark` class on the `html` element
- a small inline script in `index.html` sets the class before the first paint. This prevents a flash of the wrong theme.
- the "system" selection listens for changes to `prefers-color-scheme`
- `ThemeToggle` uses `useColorMode` from `@vueuse/core`

## Dates, numbers, and text

- the API timestamps are not RFC 3339. The UI changes the space to a `T` before it reads them.
- dates in the last 7 days show relative time, for example "3 hours ago". Older dates show "4 Nov 2024".
- scores show the maximum, for example "8 / 10"
- the UI uses `Intl` to format dates, numbers, and plurals
- the UI supports only English. Pages set `lang="en"`.

## Accessibility

- all components meet WCAG 2.2 level AA
- text and background tokens have a contrast ratio of at least 4.5:1 in both themes
- all interactive components work with the keyboard only
- focus is always visible. Components use the `ring` token for the focus outline.
- `RatingInput` uses the radio group pattern. Arrow keys change the value.
- form fields show API validation errors next to the field. The `property` of each [error](00001-define-public-api-spec.md#errors) selects the field.

## Tests

- component tests use Vitest and Vue Test Utils
- accessibility tests run `axe-core` directly on the components. `vitest-axe` has no release since 2022. Decision 00005 has the details.
- jsdom does not render, so these tests cannot check color contrast. The end-to-end tests of decision 00006 check contrast in real browsers in both themes.
- the tests run with `make test-ui`

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
