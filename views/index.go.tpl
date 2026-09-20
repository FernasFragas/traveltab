<!doctype html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>✈🌤️TravelTab</title>

    <!-- Favicon -->
    <link rel="icon" href="/traveltab.png" type="image/png">

    <!-- Bootstrap CSS -->
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.2.3/dist/css/bootstrap.min.css" rel="stylesheet"
          integrity="sha384-rbsA2VBKQhggwzxH7pPCaAqO46MgnOM80zW1RWuH61DGLwZJEdK2Kadq2F9CUG65" crossorigin="anonymous">

    <!-- Favicon -->
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap-icons@1.3.0/font/bootstrap-icons.css">

    <!-- Google Fonts -->
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@100..900&display=swap" rel="stylesheet">

    <!-- Flag Icons -->
    <link
            rel="stylesheet"
            href="https://cdn.jsdelivr.net/gh/lipis/flag-icons@7.0.0/css/flag-icons.min.css"
    />

    <link href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/5.15.4/css/all.min.css"
     rel="stylesheet"
     >

     <link rel="stylesheet" 
     href="https://cdn.jsdelivr.net/npm/bootstrap-icons/font/bootstrap-icons.css">

    <!-- Custom CSS -->
    <link rel="stylesheet" href="/styles.css">

     <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.2.3/dist/js/bootstrap.bundle.min.js"
            integrity="sha384-kenU1KFdBIe4zVF0s0G1M5b4hcpxyD9F7jL+jjXkk+Q2h455rYXK/7HAuoJl+0I4"
            crossorigin="anonymous"></script>

    <!-- HTMX Library -->
    <script src="https://unpkg.com/htmx.org@1.9.11" integrity="sha384-0gxUXCCR8yv9FM2b+U3FDbsKthCI66oH5IA9fHppQq9DDMHuMauqq1ZHBpJxQ0J0" crossorigin="anonymous"></script>

</head>
<body class="macos-bg">
    <div class="glass-container">
        <div class="container-fluid p-4">
            <div class="row justify-content-center">
                <!-- Logo -->
                <div class="col-auto text-center mb-3">
                    <img src="/traveltab.png" alt="TravelTab Logo" style="    max-width: 33.33vw; /* One third of viewport width */
                    max-height: 33.33vw;;  /* Maintain aspect ratio */
                    display: block; /* Needed for auto margins to work */
                    margin: 0 auto 1rem; /* Center horizontally, add bottom margin */
                ">
                </div>
                <!-- Search Form -->
                <div class="col-12 mb-4">
                    <div class="form-container">
                        <form hx-get="/process-form/" 
                              hx-target="#content-area" 
                              hx-swap="innerHTML" 
                              hx-indicator=".htmx-indicator">
                            <input type="text" name="city_name" placeholder="Search like Lisbon, Portugal" id="city_name" class="form-control" required>
                            <input type="submit" value="Search" class="btn btn-primary">
                            <span class="htmx-indicator ms-2">
                                <i class="fas fa-spinner fa-spin"></i>
                            </span>
                        </form>
                    </div>
                </div>

                <!-- Content Area to be updated by HTMX -->
                <div id="content-area" class="col-12">
                    <!-- Weather Display -->
                    <div class="col-12 mb-1">
                        {{ template "weather_display" . }}
                    </div>

                    <!-- Plan my trip -->
                    {{ if .Trip }}
                    <div class="col-12 mb-1">
                        {{ template "trip_card" .Trip }}
                    </div>
                    {{ end }}

                    <!-- Videos Section -->
                    <div class="col-12 mb-1">
                        {{ template "video" . }}
                    </div> 

                    <!-- Footer Section -->
                    <footer class="col-12 site-footer">
                        <p>Made with ❤️ by Fernando Fragateiro</p>
                        <div class="social-links">
                            <a href="https://github.com/FernasFragas" target="_blank" aria-label="GitHub"><i class="fab fa-github"></i></a>
                            <a href="https://pt.linkedin.com/in/fernando-paulo-fragateiro-the1" target="_blank" aria-label="LinkedIn"><i class="fab fa-linkedin"></i></a>
                            <a href="https://medium.com/@patronfragas" target="_blank" aria-label="Medium Blog"><i class="fab fa-medium"></i></a>
                            <a href="https://www.fernandofragateiro.com" target="_blank" aria-label="Personal Website"><i class="fas fa-globe"></i></a>
                            <!-- Add other social links as needed -->
                        </div>
                        <p class="data-credits">
                            Places: Wikipedia &amp; Wikidata · Photos: Wikimedia Commons · Weather: Open-Meteo (CC BY 4.0) · Map data © OpenStreetMap contributors
                        </p>
                    </footer>
                    
                 </div> <!-- End of content-area -->

            </div>
        </div>
    </div>

</body>
</html>