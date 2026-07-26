package blockchain

import (
	"testing"
)

func TestTendermintEngineProposerSelection(t *testing.T) {
	engine := NewTendermintEngine([]string{"a", "b", "c"})
	if engine.Proposer() == "" {
		t.Fatal("expected proposer")
	}
}

func TestTendermintEngineProposalPrevotePrecommit(t *testing.T) {
	engine := NewTendermintEngine([]string{"a", "b", "c"})
	block := Block{Index: 1, BlockHash: "abc"}
	_, _ = engine.ProcessProposal(block, "a")
	vote, _ := engine.Prevote("abc")
	if err := engine.AddVote(vote); err != nil {
		t.Fatalf("add prevote failed: %v", err)
	}
	vote2, _ := engine.Precommit("abc")
	if err := engine.AddVote(vote2); err != nil {
		t.Fatalf("add precommit failed: %v", err)
	}
	if err := engine.Commit("abc"); err != nil {
		t.Fatalf("commit failed: %v", err)
	}
	if !engine.IsFinalized("abc") {
		t.Fatal("block should be finalized")
	}
}

func TestTendermintEngineAdvance(t *testing.T) {
	engine := NewTendermintEngine([]string{"a", "b", "c"})
	engine.AdvanceHeight()
	if engine.Height() != 2 {
		t.Fatalf("expected height 2, got %d", engine.Height())
	}
	engine.AdvanceRound()
	if engine.CurrentRound() != 1 {
		t.Fatalf("expected round 1, got %d", engine.CurrentRound())
	}
}

func TestTendermintEngineSlashAndEvidence(t *testing.T) {
	engine := NewTendermintEngine([]string{"a", "b", "c"})
	engine.Slash("a", EvidenceDoubleSign)
	list := engine.EvidenceList()
	if len(list) != 1 {
		t.Fatalf("expected 1 evidence, got %d", len(list))
	}
}

func TestStakeWeightedSelectorDeterministic(t *testing.T) {
	vals := []string{"c", "a", "b"}
	sel := StakeWeightedSelector{}
	p0 := sel.Proposer(vals, 0)
	p1 := sel.Proposer(vals, 0)
	if p0 != p1 {
		t.Fatal("proposer selection should be deterministic")
	}
	if p0 != "a" {
		t.Fatalf("expected first proposer a, got %s", p0)
	}
}

func TestTendermintEngineCurrentState(t *testing.T) {
	engine := NewTendermintEngine([]string{"a"})
	state := engine.CurrentState()
	if state == "" {
		t.Fatal("expected non-empty state")
	}
}
