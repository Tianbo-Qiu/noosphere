# Breached password check

A breached password check refuses a new password if it has already appeared in a data breach. It works without sending the password anywhere, using a technique called k-anonymity.

## What problem it solves

Attackers do not guess passwords at random. They start with lists of passwords leaked from other services, because people reuse passwords and pick the same popular ones. A password that has leaked once is among the first an attacker tries, however long or complex it looks. [NIST SP 800-63B-4](nist-sp-800-63.md) therefore requires checking new passwords against "passwords obtained from previous breach corpuses".

## Where it is defined

- [Have I Been Pwned](https://haveibeenpwned.com/Passwords) (HIBP) runs Pwned Passwords, the most used collection of breached passwords, with a free [range API](https://haveibeenpwned.com/API/v3#PwnedPasswords): no API key, and "no rate limit".
- Cloudflare [described the k-anonymity design](https://blog.cloudflare.com/validating-leaked-passwords-with-k-anonymity/) that lets the API be used without revealing the password.
- NIST SP 800-63B-4, section 3.1.1.2, requires the check as part of a blocklist.

## Example

Looked up on 2026-10-24:

| Password | First 5 characters of its SHA-1 hash | Times seen in breaches |
|---|---|---|
| `password` | `5BAA6` | 52,372,427 |
| `Tr0ub4dor&3` | `87457` | 3,196 |
| `correct horse battery staple` | `ABF7A` | 391 |
| `quiet-otter-maps-the-river` | `99FF8` | 0 |

`Tr0ub4dor&3` looks strong, with mixed case, digits and a symbol, and a strength estimator rates it highly. It is still refused, because it has leaked thousands of times; it became famous from an [xkcd comic](https://xkcd.com/936/), and so did `correct horse battery staple`.

## How it works

1. The server hashes the password with SHA-1. `password` becomes `5BAA61E4C9B93F3F0682250B6CF8331B7EE68FD8`.
2. It sends only the first 5 characters: `GET https://api.pwnedpasswords.com/range/5BAA6`.
3. HIBP returns every leaked hash starting with that prefix, as the remaining 35 characters and a count. For `5BAA6` that was 1,977 lines, including `1E4C9B93F3F0682250B6CF8331B7EE68FD8:52372427`.
4. The server looks for its own remaining characters in the list.

Hundreds of different passwords share each prefix, so HIBP cannot tell which one was checked. With the request header `Add-Padding: true`, each response is padded to between 800 and 1,000 records with dummy entries of count 0, so its size reveals nothing either (2,159 lines for `5BAA6` with padding). SHA-1 is used only because the dataset is indexed by it; the stored password is hashed separately with [Argon2id](password-hashing.md).

The check needs the network, so a service has to decide what happens when HIBP cannot be reached: accept the password and log a warning (fail open), or refuse the sign-up until it is back (fail closed).
