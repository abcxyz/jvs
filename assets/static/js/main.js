document.addEventListener("DOMContentLoaded", function() {
    const scriptTag = document.getElementById('main');
    const targetOrigin = scriptTag.getAttribute("origin")
    const token = scriptTag.getAttribute("token")
    const windowName = scriptTag.getAttribute("windowname")

    if (!targetOrigin) {
        alert("You must pass a target origin from your application to successfully retrieve a token.")
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
 });
