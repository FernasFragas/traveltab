{{ template "trip_card" . }}
<section id="itinerary" class="tt-section" aria-labelledby="itinerary-title" tabindex="-1" hx-swap-oob="outerHTML">
    {{ template "itinerary_body" . }}
</section>
<section id="stays" class="tt-section" aria-label="Where to stay" tabindex="-1" hx-swap-oob="outerHTML">
    {{ template "stay_card" .Plan }}
</section>
