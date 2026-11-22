---
This file is not a decision.
It maps the numbers of the old decisions to the current decisions.
On 2026-11-22 the 55 decisions after 00000 became 15 decisions of one topic each.
The new decisions reuse the numbers 00001 to 00015 with other topics.
A number in an older commit message or in an old copy of a file has the old meaning. Use this table to read it.
Git keeps the old files. Read one with `git show 22124fb:docs/decisions/<file>`.
---

# Old decision numbers

| Old | Old file | Now in |
|---|---|---|
| 00001 | `00001-define-public-api-spec.md` | 00001, 00003, 00005, and 00007 |
| 00002 | `00002-add-design-system.md` | 00011 |
| 00003 | `00003-generate-sdks-from-api-spec.md` | 00001 |
| 00004 | `00004-implement-public-api-with-pocketbase.md` | 00002. The rate limits are in 00004. |
| 00005 | `00005-serve-ui-from-pocketbase-app.md` | 00002 |
| 00006 | `00006-add-end-to-end-tests.md` | 00014 |
| 00007 | `00007-validate-requests-and-responses-against-the-spec.md` | 00001 |
| 00008 | `00008-add-photos-to-reviews.md` | 00009. The upload limit is in 00004. |
| 00009 | `00009-clean-up-photo-files-without-a-record.md` | 00009 |
| 00010 | `00010-split-ci-into-jobs-with-one-result.md` | 00015 |
| 00011 | `00011-remove-tags-from-reviews.md` | 00007 |
| 00012 | `00012-index-the-overall-and-the-tags-of-restaurants.md` | 00007. The tag tables are in 00005. |
| 00013 | `00013-give-shared-links-their-own-preview.md` | 00012 |
| 00014 | `00014-add-an-overall-score-and-reviewer-pages.md` | 00007 and 00008 |
| 00015 | `00015-sign-out-on-the-server-and-sync-sessions.md` | 00003. The wrong password limit is in 00004. |
| 00016 | `00016-fix-the-dead-ends-of-restaurant-ownership.md` | 00006 |
| 00017 | `00017-add-a-location-to-restaurants.md` | 00005 and 00010 |
| 00018 | `00018-add-maps-to-the-ui.md` | 00010 |
| 00019 | `00019-simplify-the-ui.md` | 00011 |
| 00020 | `00020-fix-the-ui-items-of-the-holistic-review.md` | 00011. The map items are in 00010. |
| 00021 | `00021-add-structured-data-and-a-sitemap.md` | 00012 |
| 00022 | `00022-deploy-with-docker-compose.md` | 00013 |
| 00023 | `00023-use-rfc-3339-timestamps.md` | 00001 |
| 00024 | `00024-add-coverage-checks-and-renovate.md` | 00014 and 00015 |
| 00025 | `00025-replace-the-imaging-library.md` | 00009 |
| 00026 | `00026-run-renovate-in-a-github-action.md` | 00015 |
| 00027 | `00027-keep-the-auth-token-of-the-ui-in-an-httponly-cookie.md` | 00003 |
| 00028 | `00028-add-back-the-confirm-password-field.md` | 00003 |
| 00029 | `00029-rank-restaurants-by-a-bayesian-average.md` | 00007 |
| 00030 | `00030-add-a-cover-photo-and-a-gallery-to-restaurants.md` | 00009 |
| 00031 | `00031-index-the-overall-of-reviews.md` | 00007 |
| 00032 | `00032-add-the-average-and-the-restaurants-of-a-reviewer.md` | 00008 |
| 00033 | `00033-add-lists-of-restaurants-with-a-favorites-list.md` | 00008 |
| 00034 | `00034-require-superuser-approval-for-new-users.md` | 00003 |
| 00035 | `00035-add-a-preferred-location.md` | 00010 |
| 00036 | `00036-limit-the-write-operations.md` | 00004 |
| 00037 | `00037-add-email-confirmation-and-account-operations.md` | 00003 |
| 00038 | `00038-add-details-to-restaurants.md` | 00005 |
| 00039 | `00039-count-wrong-passwords-per-email-and-confirm-with-the-password.md` | 00003 and 00004 |
| 00040 | `00040-fix-the-spec-items-of-the-api-review.md` | 00001 |
| 00041 | `00041-allow-the-letters-of-all-scripts-in-tags.md` | 00005 |
| 00042 | `00042-get-the-current-user-without-a-new-token.md` | 00003 |
| 00043 | `00043-filter-the-restaurants-by-a-list.md` | 00008 |
| 00044 | `00044-filter-restaurants-by-the-price-level.md` | 00005 |
| 00045 | `00045-make-the-go-tests-faster.md` | 00014 |
| 00046 | `00046-make-the-end-to-end-tests-faster.md` | 00014 |
| 00047 | `00047-support-reviewing-recipes.md` | 00005 |
| 00048 | `00048-make-the-restaurant-and-recipe-apis-consistent.md` | 00001 and 00005 |
| 00049 | `00049-remove-the-structured-data-and-the-sitemap.md` | 00012 |
| 00050 | `00050-allow-a-review-for-each-visit-or-cook.md` | 00007 |
| 00051 | `00051-let-each-approved-user-edit-restaurants-and-recipes.md` | 00006 |
| 00052 | `00052-remove-the-cleanliness-score.md` | 00007 |
| 00053 | `00053-keep-all-crawlers-out-with-robots-txt.md` | 00012 |
| 00054 | `00054-rank-by-the-average-of-the-reviewers.md` | 00007 |
| 00055 | `00055-keep-a-history-of-restaurant-and-recipe-changes.md` | 00006 |
