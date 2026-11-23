# VisitorNote

Tells a signed out visitor of a shared page what Sponga47 Food is. Project component.

Lives in `@/components/app/VisitorNote.vue`, because it links to the sign up and sign in pages. Props: `name` (the reviewer or the creator of the list).

- It ends the page of a review, a journal and a list for a signed out visitor. A signed in person never sees it. See decision 00009.
- Heading in the `h3` style, `accent-foreground`: "Ana keeps food notes here".
- One sentence: "Sponga47 Food is a food journal for a small group of friends and family. Each score goes from 1 to 10, and a higher score is always better."
- Two links styled as buttons: "Create account" as the default `Button`, which is the one olive button of the page, and "Sign in" as `outline`.
- The box has the `accent` fill, `radius-2xl` and 20px padding.
