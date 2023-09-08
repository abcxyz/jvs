// Copyright 2023 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

document.addEventListener("DOMContentLoaded", async function () {
  const categorySelect = document.querySelector("#category");
  const reasonInput = document.querySelector('#reason');
  const form = document.querySelector('#form');
  const hintTooltipText = document.querySelector("#hint")

  if (!form) {
    alert("The form cannot be found");
    return;
  }

  if (!categorySelect) {
    alert("The category cell cannot be found in the form.");
    return;
  }

  if (!reasonInput) {
    alert("The reason cell cannot be found in the form.");
    return;
  }

  if(!hintTooltipText){
    alert("The hint tooltip cannot be found in the form.");
    return;
  }

  // Update the reason input's placeholder with the selected category's hint.
  function updatePlaceholder() {
    const selectedOption = categorySelect.options[categorySelect.selectedIndex];
    reasonInput.placeholder = selectedOption.getAttribute("hint");
    hintTooltipText.innerHTML = selectedOption.getAttribute("hint");
  }

  // Call the function when the page loads.
  updatePlaceholder();

  // Call the function when select new category.
  categorySelect.addEventListener("change", updatePlaceholder);

  form.addEventListener("reset", function(){
    // After resetting, the selectedIndex should be set back to 0.
    categorySelect.selectedIndex = 0;
    const defaultOption = categorySelect.options[categorySelect.selectedIndex];
    reasonInput.placeholder = defaultOption.getAttribute("hint");
  });
});

window.addEventListener("DOMContentLoaded", async () => {
  const originElement = document.querySelector("#origin");
  const windowElement = document.querySelector("#windowname");

  if (!originElement) {
    alert("The origin input was not detected.");
    window.close();
    return;
  }

  if (!windowElement) {
    alert("The windowname input was not detected.");
    window.close();
    return;
  }

  // leverage the URL parameter provided from the request and set it to the target origin
  const encodedUriComponent = new URLSearchParams(window.location.search).get("origin");
  if (!encodedUriComponent && !originElement) {
    alert("An origin URL parameter must be provided.");
    window.close();
    return;
  }

  if (encodedUriComponent) {
    const targetOrigin = decodeURIComponent(encodedUriComponent);
    if (!targetOrigin) {
      alert("Decoded URL parameter is invalid.");
      window.close();
      return;
    }

    // set values for the following hidden input elements, will be persisted to the next page
    originElement.value = targetOrigin;
    windowElement.value = window.name;
  }
}, true);
