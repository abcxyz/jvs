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

const scriptTag = document.querySelector("#success");
const targetOrigin = scriptTag.getAttribute("data-origin");
const windowName = scriptTag.getAttribute("data-window-name");
const token = scriptTag.getAttribute("data-token");
if (!targetOrigin) {
  alert("You must pass a target origin from your application to successfully retrieve a token.")
  window.close()
} else if (!windowName) {
  alert("You must pass a window name from your application to successfully retrieve a token.")
} else if (!token) {
  alert("Something went wrong, unable to retrieve a token.")
} else {
  window.opener.postMessage(
    JSON.stringify({
      // notify the requestor of the window name that was provided, 
      // client should check this as a sanity check
      source: windowName,
      payload: {
        token,
      },
    }),
    targetOrigin
  );
}
window.close();
