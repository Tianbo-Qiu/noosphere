# Password hashing

Password hashing turns a password into a fixed string that can be stored and later used to check the password, without the password itself ever being stored. It uses functions that are deliberately slow and memory-hungry, and today's recommended one is **Argon2id**.

## What problem it solves

A service has to check passwords, but storing them is dangerous: databases leak. With plain text, every account is exposed at once, and because people reuse passwords, so are their accounts elsewhere. Storing a fast hash such as SHA-256 is not much better:

- Graphics cards compute billions of SHA-256 hashes per second, so common passwords are found by trying them all.
- The same password always gives the same hash, so one precomputed table cracks every user with that password.

A password hash fixes both: each guess costs real time and memory, and a random **salt** per password makes identical passwords hash differently.

## Where it is defined

- [RFC 9106](https://www.rfc-editor.org/rfc/rfc9106) (2021) specifies Argon2. Argon2 won the [Password Hashing Competition](https://www.password-hashing.net/) (2013 to 2015), and the RFC says Argon2id must be supported by any implementation.
- The [PHC string format](https://github.com/P-H-C/phc-string-format/blob/master/phc-sf-spec.md), from the same competition, defines how a hash is written down together with its algorithm, parameters and salt.
- The [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html) gives practical settings: Argon2id with at least 19 MiB of memory, 2 iterations and 1 degree of parallelism; scrypt if Argon2id is not available; bcrypt only for legacy systems.

## Example

The password `correct horse battery staple`, stored three ways:

| Stored as | Value |
|---|---|
| SHA-256 | `c4bbcb1fbec99d65bf59d85c8cb62ee2db963f0fe106f483d9afa73bd4e39a8a` |
| Argon2id, first time | `$argon2id$v=19$m=19456,t=2,p=1$7vmMsXyFPQgp9vP7ZFhbdg$rGLcsnGSHASwcQy5t0n/1w0iSJZWCbcr/wAwP6Q0vkk` |
| Argon2id, second time | `$argon2id$v=19$m=19456,t=2,p=1$390lM5l/35sq3Kg3M+eEhQ$hOQfY9g/8eLsKi7P1kRCdH+/6Kyn1TLpoTOnMTiSoTY` |

The SHA-256 value is the same for everyone who picks this password, and takes nanoseconds to compute. The two Argon2id values differ because each has its own salt, and each took about 20 ms and 19 MiB of memory to compute.

## How it works

Argon2id fills a block of memory with values derived from the password and the salt, then reads it back in an order that depends on its contents. An attacker cannot skip the memory or trade it for speed, so the cost of each guess is set by the parameters:

| Parameter | In the PHC string | Meaning |
|---|---|---|
| Memory | `m=19456` | KiB of memory per hash (19 MiB) |
| Iterations | `t=2` | passes over that memory |
| Parallelism | `p=1` | lanes that can run in parallel |

The stored string reads as `$algorithm$version$parameters$salt$hash`, with the salt and hash in base64. To check a login, the server reads the salt and parameters from the stored string, hashes the typed password the same way, and compares the results in constant time. Because the parameters are stored with each hash, they can be raised as hardware gets faster, and old hashes are upgraded the next time their owner logs in.

Before hashing, the password is normalized with the PRECIS OpaqueString profile ([PRECIS](precis.md)), so the same password typed with a different kind of space or a decomposed accent gives the same bytes.

In Go, Argon2id is [`argon2.IDKey`](https://pkg.go.dev/golang.org/x/crypto/argon2) in `golang.org/x/crypto/argon2`.
