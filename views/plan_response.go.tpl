{{ template "trip_card" . }}
<div id="itinerary-results" hx-swap-oob="outerHTML">
    {{ template "itinerary_body" . }}
</div>
<section id="stays" class="tt-section tt-view" data-tt-view aria-label="Where to stay" tabindex="-1" hidden hx-swap-oob="outerHTML">
    {{ template "stay_card" .Plan }}
</section>
