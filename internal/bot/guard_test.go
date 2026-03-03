package bot

import (
	"testing"
	"time"
)

func TestConsumeCooldown(t *testing.T) {
	var until time.Time
	now := time.Unix(100, 0)

	remaining, ok := consumeCooldown(&until, now, 3*time.Second)
	if !ok || remaining != 0 {
		t.Fatalf("expected first consume to pass, ok=%v remaining=%v", ok, remaining)
	}

	remaining, ok = consumeCooldown(&until, now.Add(1*time.Second), 3*time.Second)
	if ok {
		t.Fatal("expected cooldown to block second consume")
	}
	if remaining <= 0 {
		t.Fatalf("expected positive remaining, got %v", remaining)
	}

	remaining, ok = consumeCooldown(&until, now.Add(4*time.Second), 3*time.Second)
	if !ok || remaining != 0 {
		t.Fatalf("expected cooldown to expire, ok=%v remaining=%v", ok, remaining)
	}
}

func TestRoomCommandGuardTryLock(t *testing.T) {
	guards := newRoomCommandGuardMap()

	state, ok := guards.tryLock("g1:c1")
	if !ok {
		t.Fatal("expected first lock to succeed")
	}
	defer state.mu.Unlock()

	if _, ok := guards.tryLock("g1:c1"); ok {
		t.Fatal("expected second lock on same room to fail while held")
	}

	state2, ok := guards.tryLock("g1:c2")
	if !ok {
		t.Fatal("expected lock on different room to succeed")
	}
	state2.mu.Unlock()
}

func TestRoomCommandGuardSnapshot(t *testing.T) {
	guards := newRoomCommandGuardMap()
	now := time.Unix(100, 0)

	s := guards.get("g1:c1")
	s.makeNextCooldown = now.Add(3 * time.Second)
	snap := guards.snapshot("g1:c1", now)
	if snap.Locked {
		t.Fatal("expected unlocked snapshot")
	}
	if !snap.CooldownKnown {
		t.Fatal("expected cooldown to be known")
	}
	if got := remainingSeconds(snap.MakeNextCooldownRemaining); got != 3 {
		t.Fatalf("expected remaining 3 seconds, got %d", got)
	}

	if _, ok := guards.tryLock("g1:c1"); !ok {
		t.Fatal("expected explicit lock for lock-state test")
	}
	snapLocked := guards.snapshot("g1:c1", now)
	if !snapLocked.Locked {
		t.Fatal("expected locked snapshot while room is locked")
	}
	if snapLocked.CooldownKnown {
		t.Fatal("expected cooldown to be unknown while locked")
	}
	s.mu.Unlock()
}
