# PRECIS

PRECIS is an IETF standard for checking whether two pieces of text, such as two usernames, should count as the same. The name stands for Preparation, Enforcement, and Comparison of Internationalized Strings.

## What problem it solves

Text that looks the same to a person can be different bytes to a computer. Uppercase letters, fullwidth letters typed with an East Asian keyboard, and invisible characters copied along with pasted text all change the bytes but not what a person sees. If a login system compares names byte by byte, three things go wrong:

- One person can create two accounts with the "same" email.
- A user who signed up as `Test@Example.com` can't log in as `test@example.com`.
- An attacker can register a copy of someone else's name that looks identical on screen.

## Where it is defined

- [RFC 8264](https://www.rfc-editor.org/rfc/rfc8264) defines the framework.
- [RFC 8265](https://www.rfc-editor.org/rfc/rfc8265) applies it to usernames and passwords.
- [RFC 8266](https://www.rfc-editor.org/rfc/rfc8266) applies it to nicknames (display names).

All three were published in 2017. They replace an older standard, [Stringprep (RFC 3454)](https://www.rfc-editor.org/rfc/rfc3454).

## Example

These inputs all look like the same email address. Here is what the PRECIS profile for usernames does with each:

| Input | Result |
|---|---|
| `test@example.com` | `test@example.com` |
| `Test@Example.com` | `test@example.com` |
| `ｔｅｓｔ@example.com` (fullwidth) | `test@example.com` |
| `test@example.com` with a zero-width space after `test` | Rejected |

The first three become the same string, so they are the same account. The last one contains a character with no legitimate use in a name, so it is refused.

## How it works

Every string is turned into one standard form before it is stored or compared. A **profile** lists the steps: map fullwidth characters to normal ones, lowercase or keep case, apply [Unicode normalization](unicode-normalization.md), then reject the string if any character is not allowed. Two strings are the same when their standard forms are identical.

RFC 8265 defines three profiles:

| Profile | For | Case |
|---|---|---|
| UsernameCaseMapped | Usernames and emails | Lowercased |
| UsernameCasePreserved | Case-sensitive usernames | Kept |
| OpaqueString | Passwords | Kept |

Passwords keep their case, because lowercasing would make `Secret` and `secret` the same password.

The same profile has to run at sign-up and at login, or the stored name will never match what the user types. PRECIS also cannot catch letters from different alphabets that look alike, such as the Cyrillic `а` and the Latin `a` ([RFC 8264, Section 12.5](https://www.rfc-editor.org/rfc/rfc8264#section-12.5)).

In Go, the profiles are in [`golang.org/x/text/secure/precis`](https://pkg.go.dev/golang.org/x/text/secure/precis).
