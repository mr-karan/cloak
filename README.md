# cloak

cloak is a utility to share senstive details like passwords, API tokens or any other secret text to a recipient. It encrypts the message with a strong [XSalsa20](https://libsodium.gitbook.io/doc/advanced/stream_ciphers/xsalsa20) cipher on client side. The key to decrypt is never stored with the backend, ensuring that only the recipient who has the link can decrypt it.

Visit https://cloak.mrkaran.dev to use.

## Self Hosting

See [docker-compose.yml](./docker-compose.yml) for an example.
