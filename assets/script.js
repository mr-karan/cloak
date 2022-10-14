
const $ = document.querySelector.bind(document);
const $show = (el) => {
    el.classList.remove('hidden');
};
const $hide = (el) => {
    el.classList.add('hidden');
};

const apiURL = '/api/';

(async function () {
    // libsodium uses Promise to signal whenever `sodium` object is ready to use.
    await window.sodium.ready;

    // Based on the URL determine whether to show encrypt or decrypt section.
    if (window.location.hash.substring(1)) {
        $hide($('.encrypt'))
        $show($('.decrypt'))
        $hide($('.works'))
    } else {
        $hide($('.decrypt'))
    }


    // Capture the form submit.
    $('#encrypt-form').onsubmit = async (e) => {
        e.preventDefault();

        const msg = $('.submit p');
        $hide(msg);

        try {
            await encrypt();
        } catch (e) {
            msg.innerText = e.toString();
            $show(msg);
            throw e;
        }
    };
})();


function encrypt() {
    const msg = $('textarea[id=password]').value,
        expiry = $('select[name=expiry]').value,
        access = $('input[name=access]').value;

    const key = sodium.from_hex('724b092810ec86d7e35c9d067702b31ef90bc43a7b598626749914d6a3e033ed');
    const nonce = sodium.randombytes_buf(sodium.crypto_secretbox_NONCEBYTES);
    const nonce_arr = sodium.to_hex(nonce);
    const enc = sodium.from_hex(nonce_arr.concat(sodium.to_hex(sodium.crypto_secretbox_easy(msg, nonce, key))));

    // Post to the API.
    fetch(apiURL + 'encrypt', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: sodium.to_hex(enc), expiry: parseInt(expiry), access_count: parseInt(access) })
    }).then(async response => {
        const data = await response.json();
        // check for error response
        if (!response.ok) {
            // get error message from body or default to response status
            const error = (data && data.message) || response.status;
            return Promise.reject(error);
        }

        $hide($('.encrypt .input'))
        $hide($('.works'))
        const shareLink = $('.share')
        $show(shareLink)
        shareLink.querySelector('#share-link').value = `${window.location.origin}/share/${data.data.uuid}#${sodium.to_hex(key)}`
    })
        .catch(error => {
            return error
        })
}

function decrypt() {
    // Get the UUID from the URL
    const uuid = window.location.pathname.split("/share/")[1]
    const hashKey = window.location.hash.substring(1)
    if (uuid === undefined) {
        throw Error("unable to parse uuid")
    }

    // Lookup for the encrypted message in backend
    fetch(apiURL + `lookup/${uuid}`, {
        method: 'GET',
        headers: { 'Content-Type': 'application/json' }
    }).then(async response => {
        const data = await response.json();
        // check for error response
        if (!response.ok) {
            // get error message from body or default to response status
            const error = (data && data.message) || response.status;
            return Promise.reject(error)
        }

        const encrypted = sodium.from_hex(data.data.message)
        if (encrypted.length < sodium.crypto_secretbox_NONCEBYTES + sodium.crypto_secretbox_MACBYTES) {
            throw Error("invalid encrypted message");
        }
        const nonce = encrypted.slice(0, sodium.crypto_secretbox_NONCEBYTES)
        const ciphertext = encrypted.slice(sodium.crypto_secretbox_NONCEBYTES)

        const decrypted = sodium.crypto_secretbox_open_easy(ciphertext, nonce, sodium.from_hex(hashKey));

        const expirySec = convertTime(data.data.expiry)

        let preText = $('#view-plaintext')
        preText.textContent = sodium.to_string(decrypted)
        $show($('.decrypt .secret-result'))
        let info = $(".secret-result .info")
        if (data.data.access_count === 0) {
            info.outerHTML = `Purged secret. This link cannot be opened again`
        } else {
            info.outerHTML = `This link is valid for <b>${expirySec}</b>. It can be opened <b>${data.data.access_count}</b> times.`
        }
        $hide($('.view-secret'))
    })
        .catch(e => {
            let errMsg = $('.decrypt .error .detail')
            errMsg.innerText = e.toString();
            $show($('.decrypt .error'));
            throw e;
        })
}

function convertTime(seconds) {
    var seconds = parseInt(seconds, 10)
    var hours = Math.floor(seconds / 3600)
    var minutes = Math.floor((seconds - (hours * 3600)) / 60)
    var seconds = seconds - (hours * 3600) - (minutes * 60)
    if (!!hours) {
        if (!!minutes) {
            return `${hours}h ${minutes}m ${seconds}s`
        } else {
            return `${hours}h ${seconds}s`
        }
    }
    if (!!minutes) {
        return `${minutes}m ${seconds}s`
    }
    return `${seconds}s`
}

function copy(selector) {
    const element = document.querySelector(selector)
    element.select()
    navigator.clipboard.writeText(element.value);
}
