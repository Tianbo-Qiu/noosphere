# Context-specific passwords

A context-specific password check refuses a password that is a word tied to the user or the service: their email address, their username, their name, or the name of the site. These are the first words an attacker who knows the account would try.

## What problem it solves

An attacker going after one account already knows a lot about it: the email address used to log in, often the person's name, and of course the site. People still pick exactly these as passwords, because they are easy to remember. A password like `Test@Example.com` for the account `test@example.com`, or `Noosphere` on noosphere, can be long enough to pass a length rule and absent from breach lists, and still fall on an attacker's first try.

## Where it is defined

[NIST SP 800-63B-4](nist-sp-800-63.md), section 3.1.1.2, requires new passwords to be compared against a blocklist that includes "context-specific words, such as the name of the service, the username, and derivatives thereof". It also says how to compare: "the entire password SHALL be subject to comparison, not substrings or words that might be contained therein". When a password is refused, the user has to be told why.

## Example

For a user signing up on noosphere as `longname@example.com`, with the first name `Bartholomew`:

| Password | Result | Why |
|---|---|---|
| `LongName@Example.com` | refused | the email, in a different case |
| `LongName` | refused | the part of the email before the `@` |
| `bartholomew` | refused | the first name |
| `Noosphere` | refused | the site's name |
| `longname@example.com and a lot more` | accepted | it contains the email, but is not the email |
| `correct horse battery staple` | accepted | not a context word (though a [breached password check](breached-password-check.md) would refuse it) |

## How it works

When the password is submitted, the service builds a short list of words from the form and from itself, and refuses the password if it equals any of them:

1. Collect the context words: the email, the part of it before the `@`, the names the user entered, and the service's name.
2. Normalize the password the same way it will be stored, for example turning a no-break space into a plain space, so a lookalike spelling cannot slip through.
3. Compare the whole password to each word, ignoring case. Only an exact match is refused.

Comparing whole words, as NIST requires, keeps the rule predictable: a user can always see why a password was refused. Some identity servers go further and refuse passwords that are merely *similar* to the email, measured by edit distance or by how much of the password is shared with the email. That catches variants like `longname@example.com1`, but it can also refuse passwords for reasons the user cannot see. Variants of common words are mostly caught by a breached password check anyway.
