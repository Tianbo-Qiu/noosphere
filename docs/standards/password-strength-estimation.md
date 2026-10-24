# Password strength estimation

Password strength estimation guesses how many attempts an attacker would need to find a password, and shows the result while the user types, usually as a meter. The most used estimator is zxcvbn, created at Dropbox.

## What problem it solves

Older strength meters counted character types: a lowercase letter, an uppercase letter, a digit and a symbol meant "strong". That rewards passwords like `Tr0ub4dor&3`, which attackers guess easily because people make the same substitutions, and punishes long passphrases of plain words. An estimator that models how attackers actually guess gives users honest feedback, which [NIST SP 800-63B-4](nist-sp-800-63.md) asks for: verifiers "SHALL offer guidance to the subscriber to help the subscriber choose a strong password".

## Where it is defined

- D. L. Wheeler, [zxcvbn: Low-Budget Password Strength Estimation](https://www.usenix.org/conference/usenixsecurity16/technical-sessions/presentation/wheeler), USENIX Security 2016, describes the method and compares it with real guessing attacks.
- [dropbox/zxcvbn](https://github.com/dropbox/zxcvbn) is the original JavaScript library; [zxcvbn-ts](https://github.com/zxcvbn-ts/zxcvbn) is its maintained TypeScript successor.

## Example

zxcvbn 4.4.2 run on 2026-10-24. The score goes from 0 (too guessable) to 4 (very unguessable):

| Password | Score | Estimated guesses | Warning |
|---|---|---|---|
| `password` | 0 | about 10^0.5 | "This is a top-10 common password" |
| `qwertyuiop` | 0 | about 10^1.4 | "This is a top-100 common password" |
| `Tr0ub4dor&3` | 4 | about 10^11 | none |
| `correct horse battery staple` | 4 | about 10^20.3 | none |
| `quiet-otter-maps-the-river` | 4 | about 10^21.8 | none |

The table also shows the limit of estimation: `Tr0ub4dor&3` scores 4, yet it appears 3,196 times in breached password lists ([Breached password check](breached-password-check.md)). An estimator judges how a password is built; only a breach check knows whether it has already leaked. The two complement each other.

## How it works

zxcvbn splits the password into the most guessable pieces it can find and adds up their cost:

- words from dictionaries of common passwords, names and English words, also when written with substitutions like `0` for `o`;
- keyboard patterns such as `qwerty` and `asdf`;
- repeats (`aaaa`), sequences (`1234`, `abcd`) and dates.

The estimate is the number of guesses an attacker following the same patterns would need, and the score buckets that number. Because its dictionaries are about 1.5 MB compressed, zxcvbn usually runs in the browser as the user types, rather than on the server.
