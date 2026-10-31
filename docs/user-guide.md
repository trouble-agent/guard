# Guard: a guide for people using products that include it

## What Guard does

Guard checks a message before a product passes it into a system that may act on it, such as an AI assistant or an automated workflow. It looks for risky or manipulative instructions and returns a handling recommendation. The product using Guard is responsible for following that recommendation.

Guard is a safety check, not a guarantee that a message is safe. A message marked for delivery can still be wrong, harmful, or misleading, and a risky message may not always be recognized.

## What the results mean

- **Deliver** means Guard did not identify a reason to hold the message. The product may continue with it. This is not a promise that the message is harmless or true.
- **Review** means the message needs extra care rather than being treated as ordinary, trusted input. The product should route it for review or handle it with reduced access. Review does not itself mean a person has reviewed or approved the message.
- **Quarantine** means the product should hold the message and not pass it on as normal. A person or the product's review process may need to decide what happens next.

The product determines what you see and what actions are available. It may show a notice, pause a workflow, or ask an authorized person to review the message.

## If Guard cannot check a message

Sometimes Guard cannot complete a check—for example, if the hosted checking service is unavailable or turned off. By default, Guard responds cautiously: it reports that the check could not be completed and the product holds the message rather than passing it on as if it had been checked.

A product may be set up to respond differently. It may let the message continue but flag it for review. An unsuccessful check is not the same as a successful check, and the product should not automatically treat the message as cleared. If a message is held or flagged, contact the product's support or review team; trying again may help if the problem was temporary.

## Privacy and external processing

Guard's hosted checking service runs outside the product's own systems. By default, the message being checked may be sent to that service so it can be assessed. Do not submit sensitive information unless the product's privacy terms and your organization's rules allow this processing.

`GUARD_EGRESS_ENABLED` is an operator setting that controls whether Guard may send messages to the hosted service. It is enabled by default. An operator can turn it off by setting it to `false`, `0`, `no`, or `off`. When it is off, Guard does not switch to a local checking service; it cannot assess messages, so the product's chosen response to a failed check applies (holding the message by default, or continuing with a review flag if configured that way). Ask the product operator which setting is in use if you need to know how your messages are processed. Turning this setting off stops Guard from sending messages to this service; it does not describe or control any other data handling performed by the product.

## What Guard does not guarantee

- Guard is not a general content-policy engine. It does not decide every question about what content is appropriate, lawful, accurate, or welcome.
- Guard cannot guarantee that it will detect every harmful instruction or that every message it allows is safe.
- Guard does not replace human judgment, an organization's safety rules, or a product's other security controls.
- A review result is not an approval, and a deliver result is not a safety certification.
- Guard's recommendation only helps when the product using it follows the result and handles errors appropriately.
