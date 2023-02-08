const urlParams = new URLSearchParams(window.location.search);

document.addEventListener("DOMContentLoaded", function () {
  // leverage the URL parameter provided from the request and set it to the target origin
  const encodedUriComponent = urlParams.get("origin");
  if (!encodedUriComponent) {
    alert("An origin URL parameter must be provided.")
    window.close();
    return;
  }

  const targetOrigin = decodeURIComponent(encodedUriComponent);
  if (!targetOrigin) {
    alert("Decoded URL parameter is invalid.")
    window.close();
    return;
  }

  // set values for the following hidden input elements, will be persisted to the next page
  document.getElementById("origin").value = targetOrigin;
  document.getElementById("windowname").value = window.name;
});
