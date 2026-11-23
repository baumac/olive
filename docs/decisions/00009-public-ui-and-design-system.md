---
description: The Public UI is a Vue app with Vue Router and shadcn-vue on Tailwind CSS. It follows the design system in specs/design-system, with one brand, light and dark themes, and accessible components. Screens start with their content, show a score as one number, and keep the forms short.
status: implemented
---

# Motivation
[motivation]: #motivation

The Public UI is the surface that the users of decision 00000 use each day.
All screens need one set of colors, spacing, and form controls. Without it, each screen makes its own choices, and accessibility problems repeat.

A calm base can still feel busy through repetition, such as rows of score dots and many controls before the content.
The users want to see a score at a glance and to fill a form fast.

Expected outcome:
- all screens use the same tokens for color, type, spacing, and radius
- all screens use the same accessible components
- light and dark themes both work. The default follows the operating system.
- the UI has one brand: a name, a logo, a palette, and app icons
- a person sees the overall score of a subject or a review at a glance
- each screen starts with its content
- the forms ask only for what the API needs

# Scope
[scope]: #scope

In scope:
- the choice of the UI framework and the design system
- the tokens, the themes, and the brand
- the component set and where it lives
- the rules for dates, numbers, and text
- the accessibility rules
- the shared layout of the screens: the header, the footer, the lists, the cards, and the score display
- the component tests

Out of scope:
- the build and the serving of the UI. Decision 00002 has them.
- the screens of each topic. The topic decisions have them.
- an icon set of our own. The UI uses Lucide.
- languages other than English, and an i18n layer
- a visual catalog of the components, such as Storybook
- sharing the design system with other projects
- a landing page

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## The stack

- the UI is written in TypeScript with Vue and Vue Router. It calls the Public API only through `@reviews/sdk`.
- the components come from [shadcn-vue](https://www.shadcn-vue.com/) on [Tailwind CSS](https://tailwindcss.com/). The CLI copies the source of each component into `app/ui/src/components/ui/`. The project owns this code.
- [Reka UI](https://reka-ui.com/) gives the accessible behavior, such as focus, keyboard control, and ARIA attributes
- `specs/design-system/README.md` is the start of the design system. It has the tokens, the component rules, the logos, and the icons.

| Folder | Content |
|---|---|
| `app/ui/src/components/ui` | the components, such as `Button`, `RatingInput`, and `MapView` |
| `app/ui/src/components/app` | the components that know the routes or call the API, such as `AppHeader`, `ErrorPage`, and `LocationField` |
| `app/ui/src/pages` | the screens |
| `app/ui/src/lib` | the helpers, such as `reviewKinds.ts`, `format.ts`, and `validation.ts` |

Screens use the components. They do not style native form controls directly.
A new component goes into the set only when a screen needs it.

## The brand

- the product name is "Sponga47 Food Reviews". The UI is at food.sponga47.com.
- the logo is an olive with a leaf, next to the name. The name uses the TeX Gyre Bonum serif as outlines. The UI text uses the system sans font.
- each logo file has a light and a dark version
- the favicon, the home screen icons, and the share image come from the logo

## Themes

- the UI has a light theme and a dark theme. The default follows `prefers-color-scheme`.
- a person selects light, dark, or system. A signed in person does it in the account menu. A signed out person does it in the page footer.
- the UI stores the selection in `localStorage`

## Layout

- the header has the logo, a "Sections" navigation with "Restaurants" and "Recipes", and the account menu
- each screen has one `h1`. The heading of a list screen can be visually hidden.
- a list screen has one toolbar row with the search, the filters, the sort, the count, the List and Map buttons, and the add button
- a card is one link. The full card opens the subject.
- a list with fewer than 10 reviews has no search, no filters, and no sort. A filter or a sort in the URL shows them.
- list filters and the sort are in the URL query. A change loads the first page again.
- a link with an old sort keeps its order. The sort control then shows it as one more option.
- a signed out visitor sees two sentences above the restaurant list. They tell what the scores are and link to the sign up page. A filter or the map view hides them.
- only the account menu and the top of the reviewer page show an avatar

## Scores

- a score shows as a number, such as "7.2", without dots and without "/ 10"
- a card shows only the overall score with the review count or the title
- the page of a subject or a review shows the overall in large type with "out of 10" one time. One line below has each score with its label.
- the label of the restaurant `price` score is "Value"

## Dates, numbers, and text

- dates in the last 7 days show relative time, such as "3 hours ago". Older dates show "4 Nov 2024".
- the `date` of a review is a local date. Decision 00005 has the labels.
- the UI uses `Intl` to format dates, numbers, and plurals
- the UI supports only English. Pages set `lang="en"`.
- `specs/design-system/formatting.md` has the rules

## Accessibility

- all components meet WCAG 2.2 level AA
- text and background tokens have a contrast ratio of at least 4.5:1 in both themes
- all interactive components work with the keyboard only
- focus is always visible as a solid 2px outline in `ring`
- `RatingInput` uses the radio group pattern. Arrow keys change the value.
- form fields show the API errors next to the field. The `property` of each error selects the field.
- the list view is the text alternative of the map view

# Implementation Details
[implementation-details]: #implementation-details

## Tokens

- the tokens are CSS custom properties in `app/ui/src/assets/main.css`. The light theme defines them on `:root`. The dark theme defines them on `.dark`.
- Tailwind CSS uses them through `@theme`. Components use classes that refer to tokens, such as `bg-primary`, not fixed colors.
- the names are the shadcn-vue names: `background`, `foreground`, `card`, `popover`, `primary`, `secondary`, `muted`, `accent`, `destructive`, `border`, `input`, `ring`, and `radius`
- the neutral colors use the olive scale of Tailwind CSS v4.2. `primary`, `ring`, and `accent` use the olive green brand color.
- `success` is blue. Blue does not look like the green brand or the red `destructive`.
- some values differ from shadcn-vue to meet the contrast rules: a darker `muted-foreground` and `input` in the light theme, a solid `ring`, and a dark `destructive` fill of 50%
- the layout tokens are the breakpoints of Tailwind CSS v4, the `content-max` and `reading-max` widths, and the `z-header`, `z-popover`, and `z-toast` layers
- `specs/design-system/tokens.json` has the values and their contrast ratios

## Themes

- the UI adds or removes the `dark` class on `html`
- `public/theme.js` sets the class before the first paint. It is a file, not an inline script, so the policy of decision 00002 allows it.
- `useTheme` has the theme state. `ThemeToggle` and `UserMenu` use it.

## Components

- the components from shadcn-vue follow the design system: a solid 2px focus outline in `ring`, the dark `destructive` fill at 50%, a `success` alert, and `radius-md` badges
- the `Form` parts are small project components, not the vee-validate parts. They set `id`, `aria-describedby`, and `aria-invalid`.
- `RatingDisplay` shows the number. It keeps `role="img"` and a name such as "Food: 8 out of 10".
- `RatingInput` links the description and the ends to the group. Hidden words make "1 means poor value. 10 means great value." for a screen reader.
- `ScoreSummary` shows the large overall and the line of scores
- `ParentCard` makes the link cover the card with an `after` element. The heart has `z-10` above it.
- `ReviewCard` is an `article`
- `FilterBar` has an `end` slot. The label of the sort is `sr-only`.
- `useListQuery` keeps the filters and the sort in the URL. It ignores unknown values, except the old sorts of `oldSortOptions`.
- the save button of the review form is outside the `form` element. Its `form` attribute joins it to the form, so the photos section is not part of the form.
- `App.vue` is a column with the minimum height of the window. The footer stays at the bottom of a short page.
- a middleware of the SDK handles a `401`. See decision 00003.
- forms save a draft in `sessionStorage` before each request that needs an account, and restore it after sign in

## Tests

- `make test-ui` runs Vitest with Vue Test Utils in jsdom
- the component tests run `axe-core` directly. `vitest-axe` has no release since 2022.
- jsdom does not render, so these tests cannot check color contrast. The end-to-end tests check contrast in real browsers in both themes. See decision 00010.

# Alternatives
[alternatives]: #alternatives

| Alternative | Result |
|---|---|
| Vuetify | Not selected. It uses Material Design, and a change of its look takes a lot of work. |
| PrimeVue | Not selected. It is large, and its styled mode is hard to change. |
| A custom design system | Not selected. Accessible selects, tabs, and dialogs take a lot of work to build and test. |
| Rows of ten dots for each score | Removed. A page with three reviews showed 24 rows that each gave one value three times. |
| Three rows of controls before the first restaurant | Removed. Each screen starts with its content. |
| A note about the price score on three pages | Removed. The label "Value" tells that a high score is good. |
| A sign up without "Confirm password" | Removed. See decision 00003. |
| An inline theme script | Not selected. The Content Security Policy refuses inline scripts. |

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
