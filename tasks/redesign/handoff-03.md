# Task 03 handoff — destination and map

Implemented the destination-only `weather_display`, current-value `weather_card`, separate `map_card`, and scoped `public/redesign/destination.css`.

- Desktop uses a 58% photo / 42% cream information split, 240px minimum height, serif city heading, optional destination-specific tagline/caption/country, and three current weather metrics.
- At 640px and below, the city overlays the photo and metrics sit on a solid surface underneath. City names wrap and can increase the photo row height.
- Nil hero metadata renders a warm neutral photo area. Failed images hide through their error handler, revealing the same fallback without moving the city or weather. Local photo credit remains available.
- Current temperature, humidity, condition text, and wave height are the reporter values. `Presentation.ConditionIcon` selects a Bootstrap icon; the template uses a thermometer fallback if absent. No weather backgrounds, blur, flag library, or Font Awesome dependency remains in these templates.
- The overview map retains `GeneralInfo.EmbedURL`, a destination-specific iframe title, Waze attribution, and the coordinate-based `Presentation.MapURL` action. The action and attribution occupy a separate 44px footer, leaving the provider's own controls unobstructed. Map frame targets approximately 242px desktop and 182px mobile including borders; content can grow.

## Verification

Real template checks covered Lisbon and a long unknown city, current weather values,
map links and iframe titles, and absent metadata. The integrated milestone passed the Go suite
and browser checks at 375, 768, 1100, and 1440px, including photo failure and long city names.
Wrapped mobile titles use a dark backing for readable contrast.

See [evidence and screenshots](../artifacts/redesign/milestone-01-03/README.md) for exact checks
and limits. Browser map checks used a labeled local fixture; live provider interaction remains unverified.

## Fidelity and external-service limits

The iframe remains a real Waze map. Waze controls its tiles, markers, colors, attribution inside the embed, and zoom behavior; parent-page CSS cannot make that content identical to the pale illustrated reference map. No decorative zoom controls or static replacement map are added. The external link and iframe attributes are verified by rendering; network availability and actual provider interaction require a live browser check and are not implied by those assertions.

The frozen presentation contract has no separate editorial-quote field, so only configured tagline, photo caption, and country are rendered. The selected photograph and its licensing/identity metadata are supplied by task 02; its exact view may differ from the reference photograph.
