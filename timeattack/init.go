// TimeAttack has implementation of mtglib.TimeAttackDetector.
package timeattack

import "time"

// DefaultDuration is a default duration when timestamps are acceptable.
//
// It means that all timestamps which are X-DefaultDuration <= X <=
// X+DefaultDuration are fine.
const DefaultDuration = 5 * time.Second
