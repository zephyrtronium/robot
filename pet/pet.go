package pet

import (
	"sync"
	"time"
)

// Status is a pet's status.
// Its methods are concurrent by way of mutual exclusion.
type Status struct {
	mu sync.Mutex

	fed, bed, kitche, living, bath, pats time.Time
}

// Satisfaction is an instantaneous view of which of a pet's needs are satisfied.
type Satisfaction struct {
	Fed, Bed, Kitche, Living, Bath, Pats bool
}

// satLocked gets the pet's satisfaction.
// The pet's mutex must be held during the call.
func (s *Status) satLocked(asof time.Time) Satisfaction {
	return Satisfaction{
		Fed:    asof.Before(s.fed),
		Bed:    asof.Before(s.bed),
		Kitche: asof.Before(s.kitche),
		Living: asof.Before(s.living),
		Bath:   asof.Before(s.bath),
		Pats:   asof.Before(s.pats),
	}
}

// Satisfaction gets the pet's satisfaction.
func (s *Status) Satisfaction(asof time.Time) Satisfaction {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.satLocked(asof)
}

// Feed feeds an amount of food to the pet, if it can eat.
// sate is interpreted as a number of minutes to add to the pet's satiation;
// if satiation is already over eight hours, then the pet will not eat.
//
// The first return value is true when the pet eats the offering.
// The second is its satisfaction after eating.
func (s *Status) Feed(asof time.Time, sate int) (bool, Satisfaction) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if asof.Add(8 * time.Hour).Before(s.fed) {
		return false, s.satLocked(asof)
	}
	if s.fed.Before(asof) {
		s.fed = asof
	}
	s.fed = s.fed.Add(time.Duration(sate) * time.Minute)
	return true, s.satLocked(asof)
}

// Room is a room that the pet needs to keep clean.
type Room int

// The pet's home is a 780 sq ft 1b1b apartment with the following layout:
//
//	┌────────┐
//	│        ├───────┐
//	│  BED   │       │
//	│           LIV  │
//	│        │       │
//	├── ─────┤       │
//	│  BATH  ├       ┤
//	│        │       │
//	├ ─────┬─┤   K   │
//	│ LAUN │C        │
//	└──────┴─┴ ──────┘
const (
	AllClean Room = iota
	Bedroom
	Kitchen
	Living
	Bathroom
)

// Clean cleans one of the pet's rooms, if any need to be cleaned.
//
// The first return value is the cleaned [Room], or [AllClean] if all were
// already clean.
// The second is its satisfaction after cleaning.
func (s *Status) Clean(asof time.Time) (Room, Satisfaction) {
	s.mu.Lock()
	defer s.mu.Unlock()
	type pair struct {
		room Room
		tm   *time.Time
	}
	ck := []pair{
		{Bedroom, &s.bed},
		{Kitchen, &s.kitche},
		{Living, &s.living},
		{Bathroom, &s.bath},
	}
	for _, c := range ck {
		if asof.Before(*c.tm) {
			continue
		}
		*c.tm = asof.Add(40 * time.Hour)
		return c.room, s.satLocked(asof)
	}
	return AllClean, s.satLocked(asof)
}

// Pat pats the pet and returns its satisfaction after patting.
// love is interpreted as a number of minutes for which the pet will feel loved
// with this pat. If the resulting time expires before its existing love, it
// has no effect.
func (s *Status) Pat(asof time.Time, love int) Satisfaction {
	s.mu.Lock()
	defer s.mu.Unlock()
	sat := asof.Add(time.Duration(love) * time.Minute)
	if s.pats.Before(sat) {
		s.pats = sat
	}
	return s.satLocked(asof)
}
