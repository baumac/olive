# UI simplicity review of 2026-11-13

This file is not a decision. It is a list of suggestions to make the Public UI simpler, from a review of `main` at bcd3eec.
A suggestion that is accepted needs a decision or a backlog item.

Method: ran `./bin/reviews-app` on a scratch database with 3 users, 3 restaurants, and 5 reviews.
Looked at the list, restaurant, review, review form, reviewer, sign in, and sign up pages. Read the component source.
The filter popover and the photo display were read in code only. They were not seen on screen.

The UI has a calm base: one color, one type stack, light shadows.
Repetition makes it feel busy. The score rows are the main cause.

Some suggestions go against `specs/design-system` as written, such as the `RatingDisplay` component and the rule of one `h1` per screen.
They need a design system update with the code.

## 1. Score display

A restaurant page with three reviews shows 24 rows of ten dots. Six are for the average, and each review card has six.
Each row gives the same value three times: the label, the dots, and "7.6 / 10".

- The dots carry no extra information. `RatingDisplay.vue` fills only whole dots, so 7.2 and 7.8 look the same. Remove them, or keep one thin bar for the overall only.
- Lead with one number. Show the overall as one large figure, with "3 reviews" next to it. Put the five scores on one muted line: "Food 8 · Service 7.7 · Space 6.7 · Value 5.7 · Clean 8".
- Cards show only the overall. The five scores belong on the page of the review. This page is already the cleanest screen.
- Remove "/ 10" after each number. State the scale one time.
- Layout bug: "Cleanliness" is wider than its label column. It touches the dots on each card.
- Rename "Price" to "Value". The note "A higher score is always better. A high price score means good value for money." then leaves three pages.

## 2. Restaurants list

- Three rows of controls come before the content: the title with the add button, then search, filters, sort, and count, then the List and Map buttons. Move the List and Map buttons into the toolbar row. Remove the "Sort by" label.
- Remove the "Restaurants" heading. The page is the home page, and the header already names it.
- The sort has seven options. Keep "Highest rated", "Newest", and "Name". Remove "Oldest", "Lowest rated", "Name, Z to A", and "Recently updated".
- Restaurant card: remove the footer with the avatar, the name, and the date of the creator. Nobody selects a restaurant by the person who added it. Make the full card the link, not only the title.

## 3. Restaurant page

- The text "Other people have reviewed this restaurant. You cannot delete it." is at the top for the owner on each visit. Move it to the edit page.
- The map is a full interactive map with zoom buttons and an "Open in OpenStreetMap" line below it. On desktop, put it in a right column next to the summary. On mobile, hide it behind the address. Make the address the link to the external map.
- The review toolbar has a search, a popover with six filters, and nine sort options. The list usually has only some reviews. Hide the toolbar below about ten reviews. Keep the sorts "Newest", "Highest", and "Lowest".
- Show the reviews as a flat list with dividers. A card with a border and a shadow for each review adds boxes.

## 4. Header

- The "Restaurants" link is the only navigation item. It goes to the same page as the logo. Remove it.
- Move the theme toggle into the account menu. Signed out visitors can get it in a footer.

## 5. Forms

- The review form has 50 buttons and ten lines of help text. Keep one short line for each score. Put the meanings of 1 and 10 at the ends of the button row.
- The photos section is below the save button. Move it above.
- Sign up: "Confirm password" repeats the job of the show password button. The API has no password reset, so a typing error loses the account. The reviewer still recommends to remove the field.

## 6. Small repeats

- Reviewer page: each card repeats the avatar and the name of the reviewer in its footer.
- The small avatars with initials on cards add a shape and no information. Keep the avatar on the reviewer page and in the account menu only.
