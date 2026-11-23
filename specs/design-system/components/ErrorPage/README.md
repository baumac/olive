# ErrorPage

A full-page message when a screen cannot show its content.

Project component. Lives in `@/components/app/ErrorPage.vue`. Props: `status` (number). It sits under `AppHeader`, so people can still navigate.

- Texts per status:
  - `404`: "We cannot find this page". "The restaurant or review may have been deleted, or the link has a typo." Actions: "Go to restaurants", "Go back".
  - `403`: "You cannot open this page". "Sign in with the account that created this record." Action: "Sign in".
  - `500` and network errors: "Something went wrong". "This is a problem on our side. Try again in a moment." Actions: "Try again", "Go to restaurants".
- The status number is small, in `primary`. The title uses `h1`. The description uses `lead` in `muted-foreground`.
- Left aligned, at most `reading-max` wide.
- The page `<title>` starts with the title, for example "We cannot find this page · Sponga47 Food".
- Use this only for the whole screen. An error inside a working screen uses `Alert` or `Toast`.
