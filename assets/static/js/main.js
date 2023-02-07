document.addEventListener("DOMContentLoaded", function() {
    const scriptTag = document.getElementById('main');
    const targetOrigin = scriptTag.getAttribute("origin")
    const token = scriptTag.getAttribute("token")
    const windowName = scriptTag.getAttribute("windowname")

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

    window.close();
 });
