package friends

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

// SECURITY regression: AcceptRequest and DeclineRequest discarded the error from
// uuid.Parse. uuid.Parse returns uuid.Nil together with the error, so a
// malformed user id silently became the all-zero UUID and the statement ran
// against the wrong identity instead of being rejected.
func TestParseFriendshipIDsRejectsMalformedIdentifiers(t *testing.T) {
	t.Parallel()

	valid := uuid.NewString()

	cases := []struct {
		name   string
		first  string
		second string
	}{
		{"malformed first", "not-a-uuid", valid},
		{"malformed second", valid, "../../etc/passwd"},
		{"empty first", "", valid},
		{"empty second", valid, " "},
		{"nil uuid first", uuid.Nil.String(), valid},
		{"nil uuid second", valid, uuid.Nil.String()},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, _, err := parseFriendshipIDs(tc.first, tc.second); !errors.Is(err, ErrInvalidUserID) {
				t.Fatalf("err = %v, want ErrInvalidUserID", err)
			}
		})
	}
}

func TestParseFriendshipIDsRejectsSelfInteraction(t *testing.T) {
	t.Parallel()

	me := uuid.NewString()
	if _, _, err := parseFriendshipIDs(me, me); !errors.Is(err, ErrCannotSelfAdd) {
		t.Fatalf("err = %v, want ErrCannotSelfAdd", err)
	}
}

func TestParseFriendshipIDsAcceptsDistinctValidIdentifiers(t *testing.T) {
	t.Parallel()

	a, b := uuid.NewString(), uuid.NewString()
	parsedA, parsedB, err := parseFriendshipIDs(a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsedA == uuid.Nil || parsedB == uuid.Nil {
		t.Fatal("valid identifiers must not resolve to the nil UUID")
	}
	if parsedA.String() != a || parsedB.String() != b {
		t.Fatalf("identifiers were reordered: got %s / %s", parsedA, parsedB)
	}
}

// Order must be preserved, not normalized: callers rely on a documented
// positional meaning, so swapping arguments has to swap the returned pair.
func TestParseFriendshipIDsPreservesArgumentOrder(t *testing.T) {
	t.Parallel()

	a, b := uuid.NewString(), uuid.NewString()

	firstAB, secondAB, err := parseFriendshipIDs(a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if firstAB.String() != a || secondAB.String() != b {
		t.Fatalf("got %s / %s, want %s / %s", firstAB, secondAB, a, b)
	}

	firstBA, secondBA, err := parseFriendshipIDs(b, a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if firstBA.String() != b || secondBA.String() != a {
		t.Fatalf("got %s / %s, want %s / %s", firstBA, secondBA, b, a)
	}
}
