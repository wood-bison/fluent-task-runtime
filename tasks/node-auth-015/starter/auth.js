import { scrypt, randomBytes, timingSafeEqual } from 'node:crypto';

/**
 * Two separate jobs that interviews love to conflate.
 *
 * 1. AUTHENTICATION — is this the right password?
 *
 *    `hashPassword(password)` returns a string safe to keep in a table that
 *    might leak. It must be salted per password, use a slow KDF (scrypt is in
 *    node:crypto), and must not contain the password.
 *    `verifyPassword(password, stored)` returns a boolean, comparing with
 *    `timingSafeEqual` — a plain `===` returns faster on an early-mismatch and
 *    that difference is measurable over a network.
 *
 * 2. AUTHORIZATION — may THIS user read THAT row?
 *
 *    `readRecord(user, recordId, records)` returns the record when the user
 *    owns it, or when `user.role === 'admin'`. Otherwise it throws an Error
 *    whose message contains "forbidden". A valid session says who you are, not
 *    what you may touch: skipping the ownership check is IDOR.
 *
 * `records` is a Map of recordId -> { ownerId, ... }.
 */
export async function hashPassword(password) {
  throw new Error('not implemented');
}

export async function verifyPassword(password, stored) {
  throw new Error('not implemented');
}

export function readRecord(user, recordId, records) {
  throw new Error('not implemented');
}
