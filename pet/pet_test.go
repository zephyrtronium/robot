package pet_test

import (
	"slices"
	"testing"
	"time"

	"github.com/zephyrtronium/robot/pet"
)

func TestSatisfaction(t *testing.T) {
	var s pet.Status
	sat := s.Satisfaction(time.Now())
	if sat != (pet.Satisfaction{}) {
		t.Errorf("surprise satisfied: %+v", sat)
	}
}

func TestFeed(t *testing.T) {
	var s pet.Status
	now := time.Now()
	ok, sat := s.Feed(now, 400)
	if !ok {
		t.Errorf("failed to feed first time")
	}
	if sat != (pet.Satisfaction{Fed: true}) {
		t.Errorf("wrong satisfied after first feed: got %+v, want only fed", sat)
	}
	ok, sat = s.Feed(now, 400)
	if !ok {
		t.Errorf("failed to feed second time")
	}
	if sat != (pet.Satisfaction{Fed: true}) {
		t.Errorf("wrong satisfied after second feed: got %+v, want only fed", sat)
	}
	ok, sat = s.Feed(now, 400)
	if ok {
		t.Errorf("fed when should have been full")
	}
	if sat != (pet.Satisfaction{Fed: true}) {
		t.Errorf("wrong satisfied after third feed: got %+v, want only fed", sat)
	}
	sat = s.Satisfaction(now.Add(800*time.Minute + 1))
	if sat != (pet.Satisfaction{}) {
		t.Errorf("satiation didn't expire: %+v", sat)
	}
	ok, sat = s.Feed(now.Add(800*time.Minute+1), 400)
	if !ok {
		t.Errorf("failed to feed final time")
	}
	if sat != (pet.Satisfaction{Fed: true}) {
		t.Errorf("wrong satisfied after final feed: got %+v, want only fed", sat)
	}
}

func TestClean(t *testing.T) {
	var s pet.Status
	now := time.Now()
	rms := make([]pet.Room, 4)
	for i := range rms {
		rms[i], _ = s.Clean(now)
	}
	want := []pet.Room{pet.Bedroom, pet.Kitchen, pet.Living, pet.Bathroom}
	slices.Sort(rms)
	slices.Sort(want)
	if !slices.Equal(rms, want) {
		t.Errorf("cleaned wrong rooms: got %v, want %v", rms, want)
	}
	r, sat := s.Clean(now)
	if r != pet.AllClean {
		t.Errorf("not all rooms were clean: %v", r)
	}
	if sat != (pet.Satisfaction{Bed: true, Kitche: true, Living: true, Bath: true}) {
		t.Errorf("wrong satisfied after cleaning: got %+v, want all rooms true", sat)
	}
	r, _ = s.Clean(now.Add(50*time.Hour + 1))
	if r == pet.AllClean {
		t.Errorf("didn't clean after clean expired")
	}
}

func TestPat(t *testing.T) {
	var s pet.Status
	now := time.Now()
	sat := s.Pat(now, 2)
	if sat != (pet.Satisfaction{Pats: true}) {
		t.Errorf("wrong satisfied after first pat: got %+v, want only pat", sat)
	}
	sat = s.Pat(now, 1)
	if sat != (pet.Satisfaction{Pats: true}) {
		t.Errorf("wrong satisfied after second pat: got %+v, want only pat", sat)
	}
	sat = s.Satisfaction(now.Add(90 * time.Second))
	if sat != (pet.Satisfaction{Pats: true}) {
		t.Errorf("wrong satisfied 90 seconds after pat: got %+v, want only pat", sat)
	}
	sat = s.Satisfaction(now.Add(121 * time.Second))
	if sat != (pet.Satisfaction{}) {
		t.Errorf("wrong satisfied 121 seconds after pat: got %+v, want none", sat)
	}
}
