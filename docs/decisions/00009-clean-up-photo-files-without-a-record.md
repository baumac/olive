---
description: Delete the photo files of review_photos that have no record with a daily PocketBase cron job.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00008 stores each photo as two files in `pb_data/storage`.
PocketBase deletes the files of a deleted record in the background, after the delete transaction.
It does not try a failed file delete again.
Some files then stay without a record:
- the file delete fails, such as from a disk error or an S3 API limit
- the server stops before the background file delete ends
- the save of a new photo fails, and the delete of its new files also fails or does not run

The Public API does not serve these files, and `/api/files` does not send them to clients.
They still use disk space.
They also keep photos that a user deleted, which goes against the privacy rules of decision 00008.

Expected outcome:
- a daily job deletes the photo folders that have no record
- the job never deletes the files of an upload that is still in progress
- superusers can see and start the job in the PocketBase dashboard

# Scope
[scope]: #scope

In scope:
- the folders of the `review_photos` collection in the PocketBase storage
- the schedule, the age rule, and the log of the job
- the tests of the job

Out of scope:
- files of other collections. No other collection has file fields.
- a file in the folder of a record that the record does not name. A photo never changes its files, so this case does not occur.
- backups of `pb_data/storage`. The backlog has this item.
- a report of the deleted files to users

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

New concepts:
- **Photo folder**: the storage folder of one `review_photos` record, `<collection id>/<record id>/`. It has the `full` and `thumb` files.
- **Orphan folder**: a photo folder whose record ID is not in `review_photos`.

The job `photo-cleanup` runs each day at 03:17 UTC. The PocketBase cron scheduler uses UTC.
It lists the files of `review_photos` in the storage and groups them by record ID.
It deletes each orphan folder whose newest file is at least 1 hour old.

The log of a run that deletes folders has a line such as:

```
INFO Photo cleanup deleted photo folders without a record folders=2
```

A failed run logs an error. The next run tries again.

Superusers see the job under Settings, Crons in the PocketBase dashboard.
They can start it there at once, for example after a disk error.
The page uses `GET /api/crons` and `POST /api/crons/{id}` of PocketBase. Both need a superuser. A test starts the job through this API.

# Implementation Details
[implementation-details]: #implementation-details

- `app/internal/photocleanup` has `Register` and `Run`. `app/main.go` calls `Register`.
- `Register` adds the job with `app.Cron().MustAdd`. PocketBase starts the cron scheduler when the server starts.
- `Run(app, now)` returns the number of deleted folders. The tests call it with a fixed time.
- the job lists the files with the PocketBase filesystem API. It then works with local storage and with a later S3 setting.
- the age rule protects an upload in progress. PocketBase writes the files of a new record before the record is in the database. The save then takes seconds, far less than 1 hour.
- the job checks the record IDs in batches of 500 with one `IN` query for each batch
- `DeletePrefix` deletes an orphan folder. A failed folder stays and adds an error. The job then continues with the other folders.
- after a failed file delete, a deleted photo stays on the disk until the next run that can delete it. When that run succeeds, the time is at most about 25 hours.
- a cause that does not go away, such as wrong storage permissions, makes each run fail for that folder. The log then has a "Photo cleanup failed" error each day, and the photo stays. A person must fix the cause.
- the job lists each file of `review_photos`, also the files of records that exist. The filesystem API of PocketBase cannot list only the folders. With 200,000 photos, the daily list has 400,000 entries. A load test can show if this matters.

Alternatives:

| Alternative | Result |
| --- | --- |
| Retry each failed file delete in a PocketBase hook | Not selected. A crash before the background delete leaves no failure to retry. |
| A queue of the files to delete in a table | Not selected. The delete of the record and the queue row would need one transaction with the file work. The scan finds all cases without new state. |
| A command for superusers instead of a cron job | Not selected. A person must then remember to run it. The dashboard can still start the cron job at once. |
| Delete without the age rule | Not selected. The job could delete the files of a photo that is in the middle of its save. |

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
