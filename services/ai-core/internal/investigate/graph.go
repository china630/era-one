package investigate

// GraphNode / GraphEdge — attack graph for Workbench (ADR-0023 AI-6).
type GraphNode struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"` // process | network | auth | other
	Label    string `json:"label"`
	EventID  string `json:"event_id,omitempty"`
}

type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Rel  string `json:"rel"`
}

type AttackGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// BuildAttackGraph links storyline steps in reverse-chronological order by category transitions.
func BuildAttackGraph(steps []StoryStep) AttackGraph {
	g := AttackGraph{}
	var prevID string
	// storyline is newest-first; reverse for causal edges
	ordered := make([]StoryStep, len(steps))
	copy(ordered, steps)
	for i, j := 0, len(ordered)-1; i < j; i, j = i+1, j-1 {
		ordered[i], ordered[j] = ordered[j], ordered[i]
	}
	for i, s := range ordered {
		id := s.EventID
		if id == "" {
			id = "step-" + s.Category + "-" + s.ObservedAt
		}
		kind := s.Category
		if kind != "process" && kind != "network" && kind != "auth" {
			kind = "other"
		}
		g.Nodes = append(g.Nodes, GraphNode{ID: id, Kind: kind, Label: s.Summary, EventID: s.EventID})
		if prevID != "" {
			g.Edges = append(g.Edges, GraphEdge{From: prevID, To: id, Rel: "followed_by"})
		}
		prevID = id
		_ = i
	}
	return g
}
