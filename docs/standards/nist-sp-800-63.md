# NIST SP 800-63

NIST SP 800-63, the *Digital Identity Guidelines*, is the US government's standard for how services should sign people up, log them in and trust logins from other systems. It is published by NIST, the National Institute of Standards and Technology, an agency of the US Department of Commerce, and is free to read.

## What problem it solves

Authentication is full of habits that feel secure but are not: forced password changes every 90 days, rules like "one uppercase letter, one digit, one symbol", security questions. Each service used to invent its own policy, and many of those policies pushed people toward weak, predictable passwords. SP 800-63 gives one evidence-based set of rules, with clear levels of strictness, that a service can follow and point to.

## Where it is defined

The current version is Revision 4, released in final form in July 2025 ([pages.nist.gov/800-63-4](https://pages.nist.gov/800-63-4/)). It has four volumes:

| Volume | Covers |
|---|---|
| [SP 800-63-4](https://pages.nist.gov/800-63-4/sp800-63.html) | Overview and assurance levels: how sure a service needs to be about who someone is |
| [SP 800-63A-4](https://pages.nist.gov/800-63-4/sp800-63a.html) | Identity proofing and enrollment: checking that a person is who they claim when they sign up |
| [SP 800-63B-4](https://pages.nist.gov/800-63-4/sp800-63b.html) | Authentication: passwords, second factors, passkeys, sessions, rate limiting |
| [SP 800-63C-4](https://pages.nist.gov/800-63-4/sp800-63c.html) | Federation: one system vouching for a user to another, as in OpenID Connect and SAML |

It is mandatory for US federal agencies. Elsewhere it is followed as the reference for good practice: the OWASP cheat sheets and many identity products base their defaults on it.

## Example

What SP 800-63B-4, section 3.1.1.2, says about passwords:

| Rule | Requirement |
|---|---|
| Minimum length | 15 characters when the password is the only factor; 8 when it is used with a second factor |
| Maximum length | allow at least 64 characters |
| Counting | "Each Unicode code point SHALL be counted as a single character" |
| Character rules | "SHALL NOT impose other composition rules (e.g., requiring mixtures of different character types)" |
| Blocklist | compare new passwords against "commonly used, expected, or compromised passwords": breached passwords, dictionary words, and context words "such as the name of the service, the username, and derivatives thereof" |
| Expiry | "SHALL NOT require subscribers to change passwords periodically", only when there is evidence of compromise |
| Guidance | "SHALL offer guidance to the subscriber to help the subscriber choose a strong password" |
| Failed attempts | limit consecutive failures to no more than 100 for one authenticator |

Several of these reverse older habits: no forced expiry, no composition rules, and a breach check instead.

## How it works

SP 800-63 sorts requirements into assurance levels, so a service picks the level that matches its risk instead of applying the strictest rules everywhere. Authentication has three levels (AAL1 to AAL3): AAL1 allows a password alone, AAL2 requires two factors, and AAL3 requires a hardware-backed factor that resists phishing. Each requirement uses the words SHALL (required), SHOULD (recommended) and MAY (allowed), as in the IETF's RFCs.

Noosphere uses volume B for its password rules and will meet volumes A and C again in the journeys on email verification and on signing in with another service.
