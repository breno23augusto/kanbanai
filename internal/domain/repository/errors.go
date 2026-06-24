package repository

import "errors"

// ErrConcurrentModification is returned by an Update when the optimistic
// locking version no longer matches the one stored. Use cases may retry the
// operation by reloading the entity and reapplying the change (SPEC §26).
var ErrConcurrentModification = errors.New("concurrent modification: version mismatch")