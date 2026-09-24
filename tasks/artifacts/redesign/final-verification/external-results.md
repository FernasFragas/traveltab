# External destination checks — 2026-09-24

These are read-only checks of the URLs the application constructs for a Lisbon trip starting
2026-09-25, with a three-night stay. They are separate from the local design-preview fixture and
do not establish that an external iframe rendered or played in a user's browser.

| Service | Exact URL checked | Observation |
| --- | --- | --- |
| Waze map | `https://embed.waze.com/iframe?zoom=10&lat=38.722300&lon=-9.139300` | Bounded `curl -L -I` returned HTTP 200, `text/html; charset=utf-8`, no redirect. The live map's visual rendering, tiles, controls, and navigation were not observed. |
| Wikimedia stop photo | `https://commons.wikimedia.org/wiki/Special:FilePath/Torre_de_Bel%C3%A9m_1.jpg?width=400` | Bounded `curl -L -I` returned HTTP 200, `image/jpeg`, after two redirects to a `thumb.wikimedia.org` 500px image. This proves this sample image endpoint responded; actual page loading and other photos were not checked. |
| YouTube privacy embed | `https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ?autoplay=1&rel=0` | Bounded `curl -L -I` returned HTTP 200, `text/html; charset=utf-8`, no redirect. HTTP response does not prove playback, autoplay permission, or video availability inside a browser iframe. The fixture's video IDs are format samples rather than an endorsed travel playlist. |
| Booking search | `https://www.booking.com/searchresults.html?ss=Lisbon%2C+pt&checkin=2026-09-25&checkout=2026-09-28` | `curl -L -I` followed one redirect to `https://www.booking.com/searchresults.html` and returned HTTP 404. A bounded GET also redirected there and returned HTTP 202 with a “verify that you're not a robot” page. Search results, prefilled destination/dates, and reservation flow therefore remain unverified. No account sign-in or reservation was attempted. |

Commands used `curl -L -I -sS --max-time 20` and, for Booking, `curl -L -sS --max-time 20 -D <temporary headers> -o <temporary body>`. Initial sandboxed requests failed DNS; the reported HTTP results are from permitted network execution outside that sandbox. An isolated headless Brave attempt to open the Booking URL did not complete within approximately 40 seconds; its process was stopped and verified absent. It produced no usable browser observation. No production server or credentials were used.

Follow-up: verify these four destinations in a normal browser session with access to the services. In particular, inspect the Booking landing page and whether the generated `pt` country code and dates survive its redirect. Treat the HTTP 202 robot challenge and HEAD 404 as an access limitation, not proof that the application's URL is wrong.
