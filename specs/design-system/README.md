The design system of the Public UI at reviews.sponga47.com. People use it to find restaurants and to review them. Later releases can review other things, so keep restaurant words out of components and put them in screens.

The system is shadcn-vue with Tailwind CSS v4, as ADR 00002 decides. The tokens use the shadcn-vue structure on the olive base color: the Tailwind olive scale, a grey with a faint green tint. The brand color is an olive green on `primary`, `ring` and `accent`. A blue `success` marks confirmations. `input` and `muted-foreground` differ from the olive pattern to meet WCAG 2.2 AA.

## Content fundamentals

- Write UI text in sentence case: "Save review", "Delete this review?", "Load more reviews".
- Button labels are a verb and an object. Avoid "OK", "Submit" and "Yes".
- Speak to the person as "you". Keep sentences short and in the present tense.
- Error text says what to do next: "Enter a title with at least one character that is not a space."
- Show data as the API stores it: tags in lowercase with hyphens (`wood-fired`), names and titles as typed.
- Name the review scores in plain words: "Space", "Service", "Food", "Value". The API calls the value score `price`. "Value" tells that a high score is good.
- No emoji and no exclamation marks.

## Visual foundations

**Color.** Olive neutrals with one olive green brand color. The page is `background` with `foreground` text. Green `primary` marks the one main action of a view, links, the selected score and the focus `ring`. Use it sparingly: one green button per view. `accent` is a green tint for hovers. Text on green is `primary-foreground`, which turns dark in the dark theme, never a literal white. `destructive` red is for deletes and errors only. Red and green are close in lightness, so a delete action always carries the word "Delete", and errors carry text and an icon. `success` is blue, not green, so a confirmation never looks like the brand or like the opposite of an error. Neutral notes use `foreground` with the info icon. Every color comes from a token through a Tailwind class, such as `bg-primary` or `text-muted-foreground`. Components never use fixed color values.

**Themes.** Light and dark use the same token names. The ui app sets them on `:root` and `.dark`. These previews use `[data-theme="dark"]` for the same purpose. The default follows `prefers-color-scheme`. A person can select light, dark or system. An inline script in `index.html` sets the class before the first paint.

**Type.** One system sans stack (`--font-sans`), no web fonts. Use the text styles in Typography: `h1` once per screen, `body` (16/28) for feedback and descriptions, `ui` (14px medium) for controls, `small` in `muted-foreground` for metadata, `caption` for badges. Use `tabular-nums` for scores.

**Spacing.** A 4px grid (`space-1` to `space-8`). Fields stack with `space-2`. Cards pad with `space-6`. Lists of cards use a `space-4` gap.

**Layout.** Mobile first, with the Tailwind breakpoints (`breakpoint-sm` 640px to `breakpoint-2xl` 1536px). Every screen has `AppHeader` on top. Page content is one centered column, at most `content-max` (1152px) wide for lists and `reading-max` (672px) for forms and long text, with 16px side padding, 24px from `sm`. Stack layers only with the `zIndex` tokens: `z-header`, `z-popover`, `z-toast`.

**Radius.** One base `radius` of 10px, with `radius-sm` 6px, `radius-md` 8px and `radius-xl` 14px derived from it. Controls are `radius-md`, cards `radius-xl`.

**Borders and shadows.** Hairlines are `border`. Control boundaries are `input`. Shadows are light: `shadow-xs` on controls, `shadow-sm` on cards, `shadow-lg` on dialogs only.

**States.** Hover darkens a fill to 90% or adds `accent`. Disabled is 50% opacity. A read-only field has the `muted` fill. Invalid fields get a `destructive` border and a `FormMessage`. Loading uses `Skeleton` on first load and a spinner in `LoadMore`.

**Focus.** Every interactive element shows a solid 2px outline in `ring` with a 2px offset when it has keyboard focus. Do not use stock shadcn's `ring/50` at 3px: it is too faint.

**Destructive fills in dark.** Delete buttons and badges in dark use `destructive` at 50% over `background` with white text (5.0:1). Stock shadcn's 60% gives 4.4:1, below 4.5:1.

**Motion.** Only the skeleton pulse, the spinner and Reka UI's open and close transitions. All of them stop under `prefers-reduced-motion`.

## Accessibility

- Components meet WCAG 2.2 level AA and work with the keyboard only.
- Text tokens hold 4.5:1 on the grounds their usage note names, in both themes.
- Control borders (`input`) and the focus outline (`ring`) hold 3:1 on every surface they sit on, in both themes.
- Status never depends on color alone. Errors have text, and alerts have an icon.

## Components

The first set covers the Public API. From shadcn-vue: `Button`, `Input`, `Textarea`, `Label`, `Form`, `Select`, `Card`, `Badge`, `Alert`, `Dialog`, `Skeleton`, `Avatar`, `Sonner` (our `Toast`) and `DropdownMenu`. Project components: `RatingInput`, `RatingDisplay`, `TagInput`, `LoadMore`, `ThemeToggle`, `UserMenu`, `SignInForm`, `SignUpForm`, `FilterBar`, `EmptyState`, `PhotoUpload`, `PhotoGrid`, `PhotoViewer`, `CoverPhoto` and `MapView`. They live in `app/ui/src/components/ui/`, one folder each. `AppHeader`, `ErrorPage`, and `KindSwitch` know the routes, and `LocationField`, `FavoriteButton`, and `SaveToListMenu` call the API, so they live in `app/ui/src/components/app/`. The Pages group shows whole pages: `SignInPage` and `SignUpPage` live in `app/ui/src/pages/`. Screens use these components and never style native controls directly. A new component goes into the set only when a screen needs it.

The previews here are static HTML renditions of the Vue components. They show the look and the states, not the behavior. The UI in `app/ui` now has the real components. Replace the previews with them with the real components: sync this system from the code with `/design-sync` in Claude Code, so the previews run the Vue components and stay in step with them.

Photos follow decision 00008. Each `<img>` gets `width` and `height` from the API, so the page does not move while it loads. Thumbnails load lazily, show `Skeleton` until they load, and show the image-off icon on `muted` when they fail. A photo without alt text is "Photo 2 of 5" for screen readers. UI text: "Add photo", "Delete this photo?", "Describe the photo for people who cannot see it". Decision 00030 adds the cover photo of restaurants and the "Photos" section of the restaurant page.

Format dates, numbers and API text as the "Dates, numbers and language" section says. Sign up, sign in, tokens and sign out follow the "User authentication" section.

## Iconography

No icon set of our own. Use `lucide-vue-next`, which shadcn-vue uses: 16px, 2px stroke, `currentColor`. The previews draw the chevron, check, close, alert and spinner shapes from Lucide. An icon-only button always has an `aria-label`.

## Logo

The logo is an olive with a leaf in `primary`, beside the name "Sponga47 Reviews" in a serif. The files and rules are in the Logos asset group. Use the light or dark file to match the theme. Use the full logo in the page header and the mark alone for the favicon and small spaces. In running text, write the name as "Sponga47 Reviews".

The App icons group has the favicons and home screen icons, made from the mark on a light green tile. The Social group has the share image that shows when someone posts a link. Its README has the `<head>` tags to copy.
