# JVS UI

**JVS is not an official Google product.**

[JVS UI](../cmd/ui) facilitates the justification verification flow using a UI. It is meant to be an alternative to using `jvsctl` which requires integrating a calling application that is non-customer facing.

## Environment Variables

The UI has the following environment variables: `PORT`, `ALLOWLIST`, and `DEV_MODE`

```shell
## default is 9091
PORT="1010" 
```

```shell
## default is false
DEV_MODE="true"
```

```shell
## A semi-colon separated string denoting the allowed domains and/or subdomains. This field is required.
ALLOWLIST="example.com;foo.bar.com"

## To allow all domain do the following
ALLOWLIST="*"
```

Setting `DEV_MODE` to `true` will automatically reload any html files without having to restart the UI server and also bypass any IP validation built within the service. If your calling application is running locally you will be able to bypass the validation without having to set this variable. 

## Run the JVS UI locally

Set your `ALLOWLIST` env variable to `*`. Run the following command from the root directory and access the UI at the port you defined above. 

```shell
go run cmd/ui/main.go
```

If you did not define a port your service should be running at `localhost:9091/popup`.

At this point there are 3 things preventing the minting of a token. 

1. The url doesn't have the expected query parameters -- `popup` and `origin`. 
2. The form does not have a "User Email" field.
3. A Cloud Key Management Service (KMS) reference to a key ring and key. 


The next section will address 1 and 2. The [KMS dependency section](#kms-dependency) will address 3. 


## Example calling application

1. To interact with your local JVS UI, you must have a calling application trigger the popup. Set up an npm directory and create run a simple express server. Ensure your `package.json` resembles the following snippet (your dependencies may be more up to date).

```json
{
  "name": "webapp",
  "version": "1.0.0",
  "description": "",
  "main": "server.js",
  "scripts": {
    "test": "echo \"Error: no test specified\" && exit 1",
    "start": "node server.js"
  },
  "keywords": [],
  "author": "",
  "license": "ISC",
  "dependencies": {
    "express": "^4.18.2"
  }
}
```

2. Run `npm i` to install dependencies for this project. 

3. Create a `server.js` at the root of the project with the following content. By default the port is set to 3000.
```javascript
const path = require("path");
const express = require("express");

const app = express();
const port = 3000;

app.use("/", express.static(path.join(__dirname, "/")));

app.listen(port, () => {
  console.log(`Sample application listening on port ${port}`);
});
```

4. Create a `index.html` at the root of the project with the following content.

```html
<html>

<head>
  <title>JVS Caller Application</title>
  <script type="text/javascript" src="helper.js"></script>
</head>

<body>
  <h2>Sample app for token retrieval</h2>
  <div>
    <button id="auth-btn">Request a token</button>
    <div id="output"></div>
  </div>
  <script type="text/javascript">
    const helper = new Helper({
      // wherever your JVS UI is running
      url: `http://localhost:9091/popup`,
      // a unique string that you should use to validate the response is indeed from JVS
      name: "auth-popup",
    });
    const button = document.querySelector("#auth-btn")
    button.addEventListener("click", async (event) => {
      try {
        // helper method to build the request and fetch the token
        const { token } = await helper.requestToken();
        document.getElementById("output").innerHTML = `Successfully retrieved token: ${token}`;
      } catch (err) {
        alert(`Failed to retrieve token: ${err}`);
      }
    });
  </script>
</body>

</html>
```

5. Create a `helper.js` at the root of the project with the following content. This file is holds the logic for building the request and handling the response from JVS.
```javascript
class Helper {
  popupRef;
  popupUrl;
  popupName;

  constructor({ url, name }) {
    // assign a name to this popup window, the JVS will send this value back in the response as an additional 
    // security measure that this calling application should validate
    this.popupName = name;
    this.popupUrl = new URL(url);

    // tells the JVS which mode to use, currently the UI supports a popup mode only
    this.popupUrl.searchParams.set("mode", encodeURIComponent("popup"));

    // tells the JVS which origin should receive the response, for non local development this value should 
    // exist in the ALLOWLIST for the JVS UI instance you are running
    this.popupUrl.searchParams.set(
      "origin",
      encodeURIComponent(window.location.origin)
    );
  }

  // receiveMessage validates the validity of the popup response origin and returns the token
  #receiveMessage = (event) => {
    // only trust the origin we opened
    if (event.origin !== this.popupUrl.origin) {
      throw new Error("invalid popup origin");
    }

    // parse the response data
    const data = JSON.parse(event.data);

    // only trust valid source variable
    if (data.source !== this.popupName) {
      throw new Error("invalid popup source");
    }

    return data.payload
  };

  // requestToken checks for a existing listeners and handles the response from the JVS UI
  requestToken = () => {
    return new Promise((resolve, reject) => {
      //remove existing event listener
      window.removeEventListener(
        "message",
        (event) => {
          try {
            resolve(this.#receiveMessage(event));
          } catch (err) {
            reject(err);
          }
        },
        false
      );

      // popup never created or was closed
      if (!this.popupRef || this.popupRef.closed) {
        this.popupRef = window.open(
          this.popupUrl.toString(),
          this.popupName,
          "popup=true,width=500,height=600"
        );
      }
      // popup exists, show it
      else {
        this.popupRef.focus();
      }

      // listen for response
      window.addEventListener(
        "message",
        (event) => {
          try {
            resolve(this.#receiveMessage(event));
          } catch (err) {
            reject(err);
          }
        },
        false
      );
    });
  };
}
```

6. Run `npm start`, your application should be running.

7. Before you click the button in the UI to trigger the JVS UI popup, you must provide an anticipated header normally provided by IAP. While running locally there is no IAP instance running so you will need to inject a header of the format `x-goog-authenticated-user-email:<your email here>`.

8. With your header set, click the button and you should not see the JVS UI with your email in the form. Provide a reason and submit the form. If you have a KMS instance running then your popup will automatically close and the token will be available in your calling application. If you dont have KMS set up see the next section. 

## KMS dependency

The UI requires a key ring and key established in your GCP project through KMS in order to successfully mint a token. Export [the `KEY` environment variable](https://github.com/abcxyz/jvs/blob/main/pkg/config/justification_config.go#L38-L40) and rerun your UI server to pick it up. 

Without KMS set up your popup should be showing an error message similar to `issue while getting key from KMS`.
