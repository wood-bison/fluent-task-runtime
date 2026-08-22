# Password storage and the ownership check

Two mistakes, both common enough to be interview staples.

**Storing something reversible.** A fast hash — md5, sha1, even sha256 —
is a poor password verifier, because it is fast for the attacker too. A slow
salted KDF and a per-password salt are what make a leaked table expensive
rather than instant, and `timingSafeEqual` keeps the comparison itself from
answering "how many bytes matched" through its duration.

**Confusing authentication with authorization.** A valid token proves who the
caller is. It says nothing about whether this record is theirs. A handler that
looks up by id and returns it is IDOR — the attacker changes the number in the
URL. The ownership check belongs in the read path, and a missing record must be
refused the same way as a forbidden one, or the error itself becomes an
existence oracle.
