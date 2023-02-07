const urlParams = new URLSearchParams(window.location.search);

document.addEventListener("DOMContentLoaded", function() {
    // leverage the URL parameter provided from the request and set it to the target origin
    const encodedUriComponent = urlParams.get("origin");
    if (!encodedUriComponent) {
        console.error("An origin URL parameter must be provided.")
        return;
    }

    const targetOrigin = decodeURIComponent(encodedUriComponent);
    if (!targetOrigin) {
        console.error("Decoded URL parameter is invalid.")
        return;
    }

    // set values for the following hidden input elements, will be persisted to the next page
    document.getElementById("origin").value = targetOrigin;
    document.getElementById("windowname").value = window.name;
 });
