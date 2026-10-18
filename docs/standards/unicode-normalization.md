# Unicode normalization

Unicode normalization is a set of standard rules for rewriting text into one agreed form, so that text which means the same thing is also stored as the same bytes. Unicode defines four of these forms: NFC, NFD, NFKC and NFKD.

## What problem it solves

Unicode often has more than one way to write the same character. `é` can be one code point (U+00E9), or the letter `e` followed by a combining accent (U+0301). Both display as `é`, but the bytes differ, so a program that compares bytes says they are not equal. Which form arrives depends on the device, the keyboard and the software that produced the text.

Without normalization:

- The same name typed on two different machines may not match.
- A search for `café` may miss a stored `café`.
- One person can register the "same" account name twice.

## Where it is defined

- [Unicode Standard Annex #15: Unicode Normalization Forms](https://www.unicode.org/reports/tr15/) defines the four forms.
- [RFC 5198](https://www.rfc-editor.org/rfc/rfc5198) asks internet protocols to exchange text in NFC.
- [PRECIS](precis.md) uses NFC as one step when comparing usernames and passwords.

## Example

| Input | NFC | NFKC |
|---|---|---|
| `é` as one code point (U+00E9) | unchanged | unchanged |
| `e` + combining accent (U+0065 U+0301) | `é` as one code point (U+00E9) | `é` as one code point (U+00E9) |
| `ｔｅｓｔ` (fullwidth) | unchanged | `test` |
| `ﬁle` (with the `ﬁ` ligature) | unchanged | `file` |
| `x²` | unchanged | `x2` |

NFC makes the two spellings of `é` identical and leaves everything else alone. NFKC goes further and also folds lookalike variants into plain letters and digits.

## How it works

The four forms come from two choices:

| | Composed (fewest code points) | Decomposed (base letter + marks) |
|---|---|---|
| **Canonical**: only characters that are truly the same | NFC | NFD |
| **Compatibility**: also variants that look alike | NFKC | NFKD |

Canonical forms never change meaning: the two kinds of `é` are the same character. Compatibility forms can lose information: `x²` becomes `x2`, which is no longer "x squared". This is why NFC is the usual choice, and why PRECIS uses NFC and handles fullwidth letters as a separate step.

Normalization only rewrites characters that Unicode considers equivalent. Invisible characters such as the zero-width space pass through unchanged, and so do letters from different alphabets that merely look alike, such as the Cyrillic `а` and the Latin `a`.

In Go, the forms are in [`golang.org/x/text/unicode/norm`](https://pkg.go.dev/golang.org/x/text/unicode/norm). Postgres has `normalize(text, NFC)` and the `IS NFC NORMALIZED` test ([string functions](https://www.postgresql.org/docs/18/functions-string.html)).
