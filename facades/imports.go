package facades

// Blank imports for packages that register Tier 3 facade types via init().
// These packages implement complex transforms (kwargs, conditional returns,
// multi-statement AST generation) that can't be expressed in facade JSON.
import (
	_ "github.com/redneckbeard/thanos/csv"
	_ "github.com/redneckbeard/thanos/net_http"
)
