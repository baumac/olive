# BottomNav

The bar at the bottom of every screen below `md`: the sections and the round "Add review".

Project component. Lives in `@/components/app/BottomNav.vue`, because it knows the routes. No props; it reads the route and the signed in user.

- A `nav` with the name "Sections" and five links: "Table" (house), "Restaurants" (utensils), "Add review" (plus), "Recipes" (chef hat) and "Pantry" (shopping basket). See decisions 00009 and 00012.
- "You" is in `UserMenu`. Six slots would be about 62px wide at 375px.
- Each link is an icon of 22px over a label in the `nav` style (11px). The current section has `aria-current="page"` and `primary`. The others are `muted-foreground`.
- "Add review" is a 52px `primary` circle with `primary-foreground` plus and `shadow-pin`, raised above the bar. It opens `/reviews/new`.
- A signed out visitor has "Table", "Restaurants", "Recipes" and "Pantry" only.
- The bar is `bottom-nav-height` (76px) with `card` fill and a `border` hairline on top, fixed at the bottom with `z-header`, above `env(safe-area-inset-bottom)`. Pages get the same bottom padding.
- From `md` the bar goes away, and `AppHeader` has the sections and "Add review".
