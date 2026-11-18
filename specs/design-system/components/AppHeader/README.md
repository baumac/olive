# AppHeader

The bar at the top of every screen: logo, sections, and account.

Project component. Lives in `@/components/app/AppHeader.vue`, outside `components/ui/`, because it knows the routes. No props; it reads the route and the signed in user.

- Left: the logo lockup, 36px high, linking to the home screen. Use the light or dark file from the Logos group to match the theme. The link's accessible name is "Sponga47 Reviews, home".
- After the logo: a `nav` with the name "Sections" and the links "Restaurants" and "Recipes". The logo still opens the restaurants list. See decision 00047.
- A section link is current on each page of its section, such as `/recipes/{id}/reviews/new`. The current link has `aria-current="page"` and the `secondary` fill. The other link is `muted-foreground` and turns `foreground` on hover.
- The section links are text links with the `md` radius. They have 8px side padding below `sm` and 12px from `sm`. The gaps of the row are 8px below `sm` and 16px from `sm`. The row then fits on a 320px screen with the mark and the account menu.
- Right: `UserMenu`. The theme is in the menu of a signed in person. A signed out person has `ThemeToggle` in the page footer.
- Height is `header-height` (64px). The content is at most `content-max` wide, with 24px side padding. It has a `border` hairline at the bottom.
- The header is sticky (`sticky top-0`) with `z-header`. Its fill is `background`, so content does not show through.
- Below `md`, swap the lockup for the olive mark alone.
- The first item in the page is a "Skip to content" link that shows on focus.
