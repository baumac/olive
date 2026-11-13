# AppHeader

The bar at the top of every screen: logo and account.

Project component. Lives in `@/components/app/AppHeader.vue`, outside `components/ui/`, because it knows the routes. No props; it reads the route and the signed in user.

- Left: the logo lockup, 36px high, linking to the home screen. Use the light or dark file from the Logos group to match the theme. The link's accessible name is "Sponga47 Reviews, home".
- It has no navigation links. The logo opens the restaurants list, and this is the only top-level screen. See decision 00019.
- Right: `UserMenu`. The theme is in the menu of a signed in person. A signed out person has `ThemeToggle` in the page footer.
- Height is `header-height` (64px). The content is at most `content-max` wide, with 24px side padding. It has a `border` hairline at the bottom.
- The header is sticky (`sticky top-0`) with `z-header`. Its fill is `background`, so content does not show through.
- Below `md`, swap the lockup for the olive mark alone.
- The first item in the page is a "Skip to content" link that shows on focus.
