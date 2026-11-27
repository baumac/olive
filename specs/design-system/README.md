The design system of Olive, a food journal for friends and family. Its Public UI is at food.sponga47.com. Its design is "The Table", a shared food journal: each visit to a restaurant and each cook of a recipe is a dated entry with photos, four scores and a few words. Restaurants and recipes are where the entries meet. Later releases can review other things, so keep restaurant words out of components and put them in screens.

The system is shadcn-vue with Tailwind CSS v4, as decision 00009 decides. The tokens use the shadcn-vue names on warm paper neutrals. The brand color is an olive green on `primary`, `ring` and `accent`. A saffron disc, the score stamp, shows the overall on photos and on the map. A blue `success` marks confirmations. The ui app has these tokens in `app/ui/src/assets/main.css`.

## Content fundamentals

- Write UI text in sentence case: "Save review", "Delete this review?", "Load more entries".
- Button labels are a verb and an object: "Add review", "Add a visit", "Add a cook", "Add restaurant". Avoid "OK", "Submit" and "Yes".
- Speak to the person as "you". Keep sentences short and in the present tense.
- Error text says what to do next: "Enter a title with at least one character that is not a space."
- Show data as the API stores it: tags in lowercase with hyphens (`wood-fired`), names and titles as typed.
- Name the review scores in plain words: "Space", "Service", "Food", "Value" for restaurants, and "Taste", "Ease", "Instructions", "Value" for recipes. The API calls the restaurant value score `price`. "Value" tells that a high score is good.
- A review is an entry in the UI. Its date reads "Visited on 14 Nov 2026" or "Cooked on 22 Nov 2026".
- The words of the journal: "On the table" for the home, "Past week" and the months for the groups, "Every visit" and "Every cook" for the reviews of a subject, "You" for the journal of the signed in person.
- No emoji and no exclamation marks.

## Visual foundations

**The idea.** The page is a menu card on a table: warm paper, ink, and one olive green. The facts of a place are calm. The photos of the food and the scores carry the color.

**Color.** The page is `background`, a warm paper, with `foreground` ink. Rows, sheets and the score block sit on `card`, a lighter paper. Olive `primary` marks the one main action of a screen, links, selected chips, the selected score and the focus `ring`. Use it sparingly: one olive button per screen, and the round "Add review" of `BottomNav`. `accent` is an olive tint for hovers, tags, the current section and `VisitorNote`. Text on olive is `primary-foreground`, which is ink in the dark theme, never a literal white. `saffron` is only for the score stamp: on photos, on journal tiles and as the map pin. `destructive` red is for deletes and errors only. Red and olive are close in lightness, so a delete action always carries the word "Delete", and errors carry text and an icon. `success` is blue, so a confirmation never looks like the brand or like the opposite of an error. Every color comes from a token through a Tailwind class, such as `bg-primary` or `text-muted-foreground`. The map pins use the `map-` tokens, which are fixed in both themes, because the tiles stay light.

**Themes.** Light and dark use the same token names. The ui app sets them on `:root` and `.dark`. These previews use `[data-theme="dark"]` for the same purpose. The default follows `prefers-color-scheme`. A person can select light, dark or system. `public/theme.js` sets the class before the first paint. Photos, map tiles and pins look the same in both themes.

**Type.** Two families. `--font-serif` is Fraunces with Georgia as the fallback: names of subjects, titles of reviews, headings and every score. `--font-sans` is the system sans stack for all other text. The ui app serves Fraunces from its own origin, as two WOFF2 files for the Latin and Latin Extended subsets with `OFL.txt` beside them, in `app/ui/src/assets/fonts`. These previews name the family, so they show Fraunces where it is installed or hosted, and Georgia otherwise. Use the styles in Typography: `h1` once per screen (`display` from `md`), `h2` for sections, `entry-title` and `name` in lists, the `Scores` styles for numbers, `reading` for the feedback of a review, `small` in `muted-foreground` for metadata, `kicker` for group labels. Scores use lining tabular figures. Text from users has `wrap-anywhere`, so a long word does not make the page wider than the screen.

**Scores.** A score is one number, such as "7.5", never dots and never "/ 10". A row, a card or an entry shows only the overall. On a photo the overall sits in the score stamp. The page of a subject or a review has the score block: the overall in `score-xl` with "out of 10" one time, and each score on its own line with its label, a dotted leader and its value, as on a menu.

**Spacing.** A 4px grid (`space-0.5` to `space-8`). Fields stack with `space-2`. Rows of a list have a `space-4` gap, entries of the table `space-5`. Page sections are `space-6` apart on a phone and `space-8` from `md`.

**Layout.** Mobile first, with the Tailwind breakpoints (`breakpoint-sm` 640px to `breakpoint-2xl` 1536px). Below `md`, `AppHeader` has the logo and the account menu, and `BottomNav` has "Table", "Restaurants", "Add review", "Recipes" and "You". From `md`, `AppHeader` has the logo, the sections, "Add review" and the account menu, and `BottomNav` goes away. Page content is one centered column, at most `content-max` (1152px) wide for lists and `reading-max` (672px) for forms and the text of a review, with 16px side padding, 24px from `sm`. From `lg`, the restaurants page shows the list and the map side by side. Stack layers only with the `zIndex` tokens: `z-header`, `z-popover`, `z-toast`.

**Radius.** Pills and circles for what a person presses: buttons, chips, the search field, the view switch, the stamp, the pins and `Avatar` (`radius-full`). Fields and dialogs have the base `radius` of 10px. Rows have `radius-xl`, photos of entries and the score block `radius-2xl`. Tags and `RatingInput` segments have `radius-md`.

**Borders and shadows.** Hairlines and leaders are `border`. Control boundaries are `input`. Shadows are few: `shadow-xs` on fields and outline buttons, `shadow-stamp` on the score stamp, `shadow-pin` on pins and the round "Add review", `shadow-md` on menus, `shadow-lg` on dialogs and toasts.

**Photos.** Photos follow decision 00007. Each `<img>` gets `width` and `height` from the API, so the page does not move while it loads. Thumbnails load lazily, show `Skeleton` until they load, and show the image-off icon on `muted` when they fail. A photo without alt text is "Photo 2 of 5" for screen readers. The photo of an entry uses the `srcset` of the `hero` size of `CoverPhoto`. The previews draw a plate on a table in place of a photo.

**States.** Hover darkens a fill to 90% or adds `accent`. Disabled is 50% opacity. A read-only field has the `muted` fill. Invalid fields get a `destructive` border and a `FormMessage`. Loading uses `Skeleton` on first load and a spinner in `LoadMore`.

**Focus.** Every interactive element shows a solid 2px outline in `ring` with a 2px offset when it has keyboard focus. Do not use stock shadcn's `ring/50` at 3px: it is too faint.

**Destructive fills in dark.** Delete buttons and badges in dark use `destructive` at 50% over `background` with `destructive-foreground` white text (6.5:1).

**Motion.** Only the skeleton pulse, the spinner and Reka UI's open and close transitions. All of them stop under `prefers-reduced-motion`.

## Accessibility

- Components meet WCAG 2.2 level AA and work with the keyboard only.
- Text tokens hold 4.5:1 on the grounds their usage note names, in both themes.
- Control borders (`input`) and the focus outline (`ring`) hold 3:1 on every surface they sit on, in both themes.
- Buttons, chips, menu items and links outside a sentence are at least 44px high (`target-min`). A link inside a sentence is exempt. `RatingInput` segments are at least 32 by 44px.
- The score stamp and the overall of a row have the hidden word "Overall" for screen readers. A map pin is a button named with the name and the overall.
- Dotted leaders and photo placeholders are hidden from screen readers.
- Status never depends on color alone. Errors have text, and alerts have an icon.
- The list view is the text alternative of the map view.

## Components

From shadcn-vue: `Button`, `Input`, `Textarea`, `Label`, `Form`, `Select`, `Card`, `Badge`, `Alert`, `Dialog`, `Skeleton`, `Avatar`, `Sonner` (our `Toast`) and `DropdownMenu`. Project components in `app/ui/src/components/ui/`, one folder each: `RatingInput`, `RatingDisplay`, `ScoreStamp`, `TagInput`, `LoadMore`, `ThemeToggle`, `UserMenu`, `SignInForm`, `SignUpForm`, `FilterBar`, `EmptyState`, `PhotoUpload`, `PhotoGrid`, `PhotoViewer`, `CoverPhoto` and `MapView`. `AppHeader`, `BottomNav`, `EntryCard`, `ScoreSummary`, `VisitorNote`, `ErrorPage` and `KindSwitch` know the routes or the kinds, and `LocationField`, `FavoriteButton`, and `SaveToListMenu` call the API, so they live in `app/ui/src/components/app/`. The Pages group shows whole pages: `SignInPage` and `SignUpPage` live in `app/ui/src/pages/`. Screens use these components and never style native controls directly. A new component goes into the set only when a screen needs it.

The previews here are static HTML renditions of the Vue components. They show the look and the states, not the behavior. The components are in `app/ui`. Sync this system from the code with `/design-sync` in Claude Code, so the previews run the Vue components and stay in step with them. The design canvas "The Table: a food journal redesign" has the screens these components make.

Format dates, numbers and API text as the "Dates, numbers and language" section says. Sign up, sign in, tokens and sign out follow the "User authentication" section.

## Iconography

No icon set of our own. Use `lucide-vue-next`, which shadcn-vue uses: 16px, 2px stroke, `currentColor`. `BottomNav` uses 22px icons: house for "Table", utensils for "Restaurants", plus for "Add review", chef hat for "Recipes" and book open for "You". The kinds use utensils for restaurants and chef hat for recipes everywhere. An icon-only button always has an `aria-label`.

## Logo

The logo is an olive with a leaf in `primary`, beside the name "Olive" in a serif. The files and rules are in the Logos asset group. Use the light or dark file to match the theme. Use the full logo in the page header from `md`. Below `md`, and for the favicon and small spaces, use the mark alone. Do not set the name in Fraunces next to the mark: the logo files have the name as outlines. In running text, write the name as "Sponga47 Food".

The App icons group has the favicons and home screen icons, made from the mark on a light green tile. The Social group has the share image that shows when someone posts a link. Its README has the `<head>` tags to copy.
