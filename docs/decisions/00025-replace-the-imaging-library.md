---
description: The photo upload decodes and resizes images with the Go standard library and golang.org/x/image, not with disintegration/imaging.
status: implemented
---

# Motivation
[motivation]: #motivation

The photo upload of decision 00008 used `github.com/disintegration/imaging` to decode, turn, and resize each upload.
Its last release, v1.6.2, is from 2019. The uploads are data that the app cannot trust.
A library without releases gets no fixes.

Expected outcome:
- the code that reads uploads comes from the Go project, which has releases and security fixes

# Scope
[scope]: #scope

In scope:
- the decode, the EXIF orientation, and the resize in `app/internal/publicapi`

Out of scope:
- the limits, the output sizes, and the JPEG quality of decision 00008
- the `imaging` module in `app/go.sum`. PocketBase needs it for its own thumbnails. The app does not make these thumbnails.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

The API does not change. A photo has the same sizes as before.
The pixels can be a little different, because the resize filter changed from Lanczos to Catmull-Rom.

# Implementation Details
[implementation-details]: #implementation-details

- `image.Decode` of the standard library decodes JPEG and PNG. `golang.org/x/image/webp` decodes WebP, as before.
- `exifOrientation` in `photoorient.go` reads the orientation tag of a JPEG file. It reads only the first IFD of the EXIF segment.
  It checks each offset against the length of the data. A file that it cannot read has the orientation 1.
- `fit` makes the image smaller with `draw.CatmullRom` of `golang.org/x/image/draw`. It does not make a small image larger.
- `orient` turns the pixels after the fit. It then moves at most 2048 by 2048 pixels.
- PNG and WebP files keep their pixels as they are. `imaging` also read the orientation only from JPEG files.
- the resize uses one CPU core. An image of 6000 by 4000 pixels takes about 0.6 seconds on the test computer. `imaging` used all cores.
  The semaphore of decision 00008 already limits the decodes that run at the same time.

| Option | Result |
| --- | --- |
| the standard library and `golang.org/x/image` | Selected. No new module. The Go project maintains both. |
| a maintained fork of `imaging` | Not selected. A fork of one person can stop like the original. |
| `libvips` through cgo | Not selected. The build then needs a C compiler and a C library in the image. |

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
