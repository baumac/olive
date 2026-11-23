---
description: The Public UI is a Vue app with Vue Router and shadcn-vue on Tailwind CSS. Its design is "The Table", a shared food journal. Each visit and each cook is a dated entry, and restaurants and recipes are where the entries meet. The UI has warm paper colors, the Fraunces serif for names and scores, one brand, light and dark themes, and accessible components.
status: implemented
---

# Motivation
[motivation]: #motivation

The Public UI is the surface that the users of decision 00000 use each day.
Decision 00000 wants to save, review, and share food experiences.
A list of restaurants is a directory. It does not show the experiences.
Each review is one visit or one cook with its own date. See decision 00005.
The UI therefore puts the experience first and the subject second.

All screens need one set of colors, type, spacing, and form controls. Without it, each screen makes its own choices, and accessibility problems repeat.

A calm base can still feel busy through repetition, such as rows of score dots and many controls before the content.
The users want to see a score at a glance and to fill a form fast.
Most visitors arrive through a shared link. The page of a review or a list is often their first page of the app.

Expected outcome:
- the home shows the latest reviews of all users, the newest written first
- a user adds a review from each screen with one button, and selects or adds the restaurant or the recipe in the same form
- each reviewer has a journal of the own entries by month
- a shared review or list tells a visitor without an account what the app is
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
- the tokens, the themes, the type, and the brand
- the component set and where it lives
- the rules for dates, numbers, and text
- the accessibility rules
- the navigation and the home
- the shared layout of the screens: the header, the bottom bar, the footer, the lists, the entries, the cards, and the score display
- the note for visitors on shared pages
- the component tests

Out of scope:
- the build and the serving of the UI. Decision 00002 has them.
- the screens of each topic. The topic decisions have them.
- an icon set of our own. The UI uses Lucide.
- languages other than English, and an i18n layer
- a visual catalog of the components, such as Storybook
- sharing the design system with other projects
- a landing page
- follows, a feed of followed users, likes, comments, and notifications. Decision 00000 puts social features out of scope. The backlog has them under "Won't do".
- a share button. A person shares the link of the page.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

## The concepts

- **Entry**: one review, shown as one experience. It has the photo, the subject, the title, the reviewer, the date, and the overall. The API calls it a review. See decision 00005.
- **Table**: the home at `/`. It shows the latest entries of all users, the newest written first.
- **Journal**: the page of one reviewer. It has the entries by month, the favorites, and the lists. See decision 00006.
- **Score stamp**: the overall in a round saffron disc. It sits on photos and is the pin of the maps.
- **Score block**: the large overall with each score on a line, as on a menu.

Examples:
- Ana opens the app. The table shows that Priya visited Kin Khao Noodle Bar two days ago and gave it 8.8.
- Ana selects "Add review" in the bottom bar, then "I ate out". She selects Luigi's Trattoria, "Yesterday", four scores, and a title.
- Ana sends the link of her review to a friend. The friend sees the photo, the score block, and the feedback. A note at the end tells what Sponga47 Food is.

## The stack

- the UI is written in TypeScript with Vue and Vue Router. It calls the Public API only through `@reviews/sdk`.
- the components come from [shadcn-vue](https://www.shadcn-vue.com/) on [Tailwind CSS](https://tailwindcss.com/). The CLI copies the source of each component into `app/ui/src/components/ui/`. The project owns this code.
- [Reka UI](https://reka-ui.com/) gives the accessible behavior, such as focus, keyboard control, and ARIA attributes
- `specs/design-system/README.md` is the start of the design system. It has the tokens, the component rules, the logos, and the icons.

| Folder | Content |
|---|---|
| `app/ui/src/components/ui` | the components, such as `Button`, `RatingInput`, `ScoreStamp`, and `MapView` |
| `app/ui/src/components/app` | the components that know the routes or call the API, such as `AppHeader`, `BottomNav`, `EntryCard`, and `LocationField` |
| `app/ui/src/pages` | the screens |
| `app/ui/src/lib` | the helpers, such as `reviewKinds.ts`, `format.ts`, and `validation.ts` |

Screens use the components. They do not style native form controls directly.
A new component goes into the set only when a screen needs it.

## The brand

- the product name is "Sponga47 Food". The UI is at food.sponga47.com.
- the logo is an olive with a leaf, next to the name. The name uses the TeX Gyre Bonum serif as outlines.
- each logo file has a light and a dark version
- the favicon, the home screen icons, and the share image come from the logo

## Color and type

- the page is warm paper with ink text, like a menu card. Cards are a lighter paper.
- olive green marks the one main action of a screen, the links, the selected controls, and the focus ring
- saffron is only for the score stamp
- red is only for deletes and errors. Blue is for confirmations.
- names, titles, headings, and scores use the Fraunces serif. All other text uses the system sans font.
- buttons, chips, and the search field are pills. Cards and photos have round corners.

## Themes

- the UI has a light theme and a dark theme. The default follows `prefers-color-scheme`.
- a person selects light, dark, or system. A signed in person does it in the account menu. A signed out person does it in the page footer.
- the UI stores the selection in `localStorage`
- photos, the map tiles, and the map pins look the same in both themes

## Navigation

- below the `md` breakpoint, a bottom bar has "Table", "Restaurants", "Add review", "Recipes", and "You". "Add review" is a round olive button in the middle.
- below `md`, the header has the logo and the account menu. The account menu has the account page, the theme, and "Sign out".
- from `md`, the header has the logo, "Table", "Restaurants", "Recipes", the "Add review" button, and the account menu. The account menu has "You".
- a signed out visitor has no "Add review" and no "You". The header has "Sign in".
- `/` opens the table. `/restaurants` and `/recipes` open the lists of the subjects.
- "You" opens the journal of the signed in user. See decision 00006.
- each screen has one `h1`. The heading of a list screen can be visually hidden.

## The table

- the heading is "On the table", with the day above it and "What everyone ate and cooked lately." below it
- "All", "Ate out", and "Cooked" select the kinds. `?kind=restaurants` and `?kind=recipes` keep the selection in the URL.
- the groups are "Past week", then the months, such as "Earlier in November" and "October 2026". The group comes from `created`, the time the review was written. A late review of an old dinner is then at the top, with its own "Visited on" date.
- an entry with photos shows the first photo with the score stamp. An entry without photos is a short row with the overall on the right.
- an entry shows the subject, the title, the reviewer, and "Visited on 20 Nov 2026" or "Cooked on 22 Nov 2026"
- "Load more entries" loads the next entries
- a signed out visitor sees two sentences above the entries. They tell what the scores are and link to the sign up page.

## Lists of subjects

- a list screen has one toolbar with the search, the filters, the sort, the count, the List and Map buttons, and "Add restaurant" or "Add recipe". On a phone, it wraps to two rows.
- a row of a subject is one link. It has the cover, the name, the distance or the total time, the review count, the tags, and the overall. The heart is a separate button next to it.
- a list of reviews with fewer than 10 reviews, such as the timeline of a subject, has no search, no filters, and no sort. A filter or a sort in the URL shows them.
- list filters and the sort are in the URL query. A change loads the first page again.
- a link with an old sort keeps its order. The sort control then shows it as one more option.
- only the account menu and the top of the journal show an avatar

## Scores

- a score shows as a number, such as "7.2", without dots and without "/ 10"
- an entry, a row, or a card shows only the overall
- on a photo, the overall is in the score stamp
- the page of a subject or a review has the score block. The overall is in large type with "out of 10" one time. Each score is on its own line with its label, a dotted leader, and its value.
- the label of the restaurant `price` score is "Value"

## Shared pages

- the page of a review or a list ends with a note for a signed out visitor
- the note has a heading such as "Ana keeps food notes here", one sentence about the app and the scores, "Create account", and "Sign in"
- the pages of a restaurant, a review, a journal, and a list have link previews. See decision 00002.

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
- buttons, chips, and links outside a sentence are at least 44 px high. A link inside a sentence, such as the name of the reviewer, is exempt, as WCAG 2.2 allows.
- focus is always visible as a solid 2px outline in `ring`
- `RatingInput` uses the radio group pattern. Arrow keys change the value.
- the score stamp and the overall of a row have the hidden word "Overall" for screen readers
- the dotted leaders and the photo placeholders are hidden from screen readers
- form fields show the API errors next to the field. The `property` of each error selects the field.
- the list view is the text alternative of the map view

# Implementation Details
[implementation-details]: #implementation-details

## Tokens

- the tokens are CSS custom properties in `app/ui/src/assets/main.css`. The light theme defines them on `:root`. The dark theme defines them on `.dark`.
- Tailwind CSS uses them through `@theme`. Components use classes that refer to tokens, such as `bg-primary`, not fixed colors.
- the names are the shadcn-vue names: `background`, `foreground`, `card`, `popover`, `primary`, `secondary`, `muted`, `accent`, `destructive`, `border`, `input`, `ring`, and `radius`. `saffron`, `saffron-foreground`, and `success` are the project names.
- the table below has the values and their contrast ratios. `specs/design-system/tokens.json` has the same values. The ui app gets them when this design is built.

| Token | Light | Dark | Contrast |
|---|---|---|---|
| `background` | `#F6F2E8` | `#15140F` | |
| `card`, `popover` | `#FFFDF7` | `#201E17` | |
| `foreground` | `#1D1F14` | `#F2EDE0` | 14.9:1 and 15.8:1 on `background` |
| `card-foreground`, `popover-foreground` | `#1D1F14` | `#F2EDE0` | 16.4:1 and 14.3:1 on `card` |
| `muted-foreground` | `#585A48` | `#B3AE9C` | 6.3:1 and 8.3:1 on `background` |
| `secondary`, `muted` | `#EEE9DC` | `#2A2820` | `muted-foreground` 5.8:1 and 6.6:1 on it |
| `secondary-foreground` | `#1D1F14` | `#F2EDE0` | 13.8:1 and 12.6:1 on `secondary` |
| `primary`, `ring` | `#4B6113` | `#B7C878` | 6.2:1 and 10.1:1 on `background` |
| `primary-foreground` | `#FBFAF3` | `#1D1F14` | 6.7:1 and 9.2:1 on `primary` |
| `accent` | `#E8ECD4` | `#2B3119` | |
| `accent-foreground` | `#3D4F0E` | `#CFDD95` | 7.5:1 and 9.3:1 on `accent` |
| `border` | `#E2DCCB` | `#36332A` | hairlines and leaders only |
| `input` | `#878672` | `#7C7868` | at least 3:1 on `background`, `card`, and `muted` |
| `saffron` | `#F0BD45` | `#E8B744` | |
| `saffron-foreground` | `#1D1F14` | `#1D1F14` | 9.6:1 and 9.0:1 on `saffron` |
| `destructive` | `#B3261E` | `#F28B7F` | 5.9:1 and 7.7:1 on `background` |
| `destructive-foreground` | `#FFFFFF` | `#FFFFFF` | 6.5:1 on the `destructive` fill, and 6.5:1 on the dark 50% fill |
| `success` | `#1F5FA8` | `#8DB8F0` | 5.8:1 and 9.0:1 on `background` |
| `success-foreground` | `#FBFAF3` | `#1D1F14` | 6.2:1 and 8.1:1 on `success` |

- `success` is blue. Blue does not look like the green brand or the red `destructive`.
- the dark `destructive` fill is 50% over `background`. White text on it has 6.5:1.
- the layout tokens are the breakpoints of Tailwind CSS v4, the `content-max` and `reading-max` widths, and the `z-header`, `z-popover`, and `z-toast` layers. The bottom bar uses `z-header`.

## Type

- `--font-serif` is Fraunces with the fallback Georgia. `--font-sans` is the system sans stack.
- Fraunces is a variable WOFF2 font with the `opsz` and `wght` axes. It has two files, the Latin and the Latin Extended subsets, with a `unicode-range` each. A name such as `İstanbul` then uses one font.
- the files are in `app/ui/src/assets/fonts`, with `OFL.txt`, the SIL Open Font License of Fraunces
- the app serves the font from its own origin. The Content Security Policy of decision 00002 then needs no new source.
- `@font-face` has `font-display: swap`. Text shows in Georgia until the font loads.
- scores use `lining-nums tabular-nums`

## Themes

- the UI adds or removes the `dark` class on `html`
- `public/theme.js` sets the class before the first paint. It is a file, not an inline script, so the policy of decision 00002 allows it.
- `useTheme` has the theme state. `ThemeToggle` and `UserMenu` use it.

## Components

- the components from shadcn-vue follow the design system: a solid 2px focus outline in `ring`, the dark `destructive` fill at 50%, a `success` alert, pill buttons, and `radius-md` badges
- the `Form` parts are small project components, not the vee-validate parts. They set `id`, `aria-describedby`, and `aria-invalid`.
- `BottomNav` is a `nav` with the name "Sections". The current section has `aria-current="page"`. It sits above `env(safe-area-inset-bottom)`, and the pages get a bottom padding of its height.
- `EntryCard` has two variants: `photo` with the first photo and the score stamp, and `row` without a photo. The card is one link.
- `ScoreStamp` shows the overall in a saffron disc with `saffron-foreground` text. It has the sizes `sm` and `md`. `MapView` uses the same look for its pins.
- `RatingDisplay` shows the number. It keeps `role="img"` and a name such as "Food: 8 out of 10".
- `RatingInput` links the description and the ends to the group. Hidden words make "1 means poor value. 10 means great value." for a screen reader. Each of the ten segments is at least 32 by 44 px.
- `ScoreSummary` is the score block. It is a `dl` with a `dt` and a `dd` for each score. The leader is an `aria-hidden` element between them.
- `VisitorNote` is the note for a signed out visitor on shared pages
- `KindSwitch` has the values "All", restaurants, and recipes. Each value is a link, so each has its own URL.
- `ParentCard` makes the link cover the row with an `after` element. The heart has `z-10` above it.
- `ReviewCard` is an `article`
- `FilterBar` has an `end` slot. The label of the sort is `sr-only`.
- `useListQuery` keeps the filters and the sort in the URL. It ignores unknown values, except the old sorts of `oldSortOptions`.
- the save button of the review form is outside the `form` element. Its `form` attribute joins it to the form, so the photos section is not part of the form.
- `App.vue` is a column with the minimum height of the window. The footer stays at the bottom of a short page.
- a middleware of the SDK handles a `401`. See decision 00003.
- forms save a draft in `sessionStorage` before each request that needs an account, and restore it after sign in

## The table

- `useEntries` loads `listAllRestaurantReviews` and `listAllRecipeReviews` with `sort=-created`. Each source keeps its own cursor.
- it merges the loaded entries by the key of the sort, the newest first. The table uses `-created`. The journal uses `-date`, which sorts by `date`, then `created`.
- with both sources, the order is always this key. The sort control and the score filters need one kind.
- the merged list ends where a source runs out of loaded entries but has more pages. Each shown entry is then in its final place.
- "Load more entries" loads the next page of each source that has run out
- with `?kind=`, it loads only one source
- the journal of decision 00006 uses `useEntries` with the reviewer lists

## Tests

- `make test-ui` runs Vitest with Vue Test Utils in jsdom
- the component tests run `axe-core` directly. `vitest-axe` has no release since 2022.
- jsdom does not render, so these tests cannot check color contrast. The end-to-end tests check contrast in real browsers in both themes. See decision 00010.
- the tests of `useEntries` check the merge order, a tie of dates, and an empty source

# Alternatives
[alternatives]: #alternatives

| Alternative | Result |
|---|---|
| Vuetify | Not selected. It uses Material Design, and a change of its look takes a lot of work. |
| PrimeVue | Not selected. It is large, and its styled mode is hard to change. |
| A custom design system | Not selected. Accessible selects, tabs, and dialogs take a lot of work to build and test. |
| The restaurant list as the home | Removed. A directory hid the experiences. The table shows them by date, and the Restaurants section keeps the list. |
| White pages with only the system sans font | Removed. The warm paper and a serif like the logo fit a food journal better. |
| A receipt and kitchen ticket style | Not selected. Uppercase names and handwritten titles are harder to read, and the style needs two web fonts. |
| Fraunces from Google Fonts | Not selected. The Content Security Policy allows only the own origin, and Google would get the IP address of each visitor. |
| A system serif such as Georgia | Not selected. It differs between devices, and so do the numerals of the scores. |
| Rows of ten dots for each score | Removed. A page with three reviews showed 24 rows that each gave one value three times. |
| Three rows of controls before the first restaurant | Removed. Each screen starts with its content. |
| A note about the price score on three pages | Removed. The label "Value" tells that a high score is good. |
| A sign up without "Confirm password" | Removed. See decision 00003. |
| An inline theme script | Not selected. The Content Security Policy refuses inline scripts. |
| The name "Sponga47 Reviews" at reviews.sponga47.com | Removed. The app is a shared food journal, not only reviews. "Sponga47 Food" at food.sponga47.com tells this. |
| The tagline "Find a restaurant. Score it. Say why." | Removed. It names only restaurants. The table has the cooks of recipes too. |

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
