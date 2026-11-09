# AppHeader

The bar at the top of every screen: logo, main navigation, theme and account.

Project component. Lives in `@/components/app/AppHeader.vue`, outside `components/ui/`, because it knows the routes. No props; it reads the route and the signed in user.

- Left: the logo lockup, 36px high, linking to the home screen. Use the light or dark file from the Logos group to match the theme. The link's accessible name is "Sponga47 Reviews, home".
- Middle: `nav` with `aria-label="Main"`. One link: "Restaurants". The current page has `aria-current="page"` and an `accent` fill.
- Right: `ThemeToggle`, then `UserMenu`.
- Height is `header-height` (64px). The content is at most `content-max` wide, with 24px side padding. It has a `border` hairline at the bottom.
- The header is sticky (`sticky top-0`) with `z-header`. Its fill is `background`, so content does not show through.
- Below `md`, swap the lockup for the olive mark alone, hide the nav links and show a menu button that opens them in a `Sheet`. Add `Sheet` to the set when this screen is built.
- The first item in the page is a "Skip to content" link that shows on focus.
