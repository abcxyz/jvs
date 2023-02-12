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

const urlParams = new URLSearchParams(window.location.search);

window.addEventListener("DOMContentLoaded", async () => {
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
}, true);
