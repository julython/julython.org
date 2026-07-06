package metrics

// CalculateLevel maps a 0–10 heuristic score to a 0–3 level:
// 0 = no data, 1–5 = L1, 6–8 = L2, 9–10 = L3.
func CalculateLevel(score int16) int16 {
	switch {
	case score >= 9:
		return 3
	case score >= 6:
		return 2
	case score >= 1:
		return 1
	default:
		return 0
	}
}

// Evaluate builds the eight metric rows from a completed scan (tree + file contents).
func Evaluate(res *ScanResult) map[string]MetricResult {
	lang := res.Language

	out := make(map[string]MetricResult, 8)

	r, rd, _ := evalReadme(res)
	setLanguage(rd, lang)
	out["readme"] = MetricResult{Data: rd, Score: Score(r)}

	t, td, _ := evalTests(res)
	setLanguage(td, lang)
	out["tests"] = MetricResult{Data: td, Score: Score(t)}

	c, cd, _ := evalCI(res)
	setLanguage(cd, lang)
	out["ci"] = MetricResult{Data: cd, Score: Score(c)}

	s, sd, _ := evalStructure(res)
	setLanguage(sd, lang)
	out["structure"] = MetricResult{Data: sd, Score: Score(s)}

	l, ld, _ := evalLinting(res)
	setLanguage(ld, lang)
	out["linting"] = MetricResult{Data: ld, Score: Score(l)}

	d, dd, _ := evalDeps(res)
	setLanguage(dd, lang)
	out["deps"] = MetricResult{Data: dd, Score: Score(d)}

	do, ddo, _ := evalDocs(res)
	setLanguage(ddo, lang)
	out["docs"] = MetricResult{Data: ddo, Score: Score(do)}

	a, ad, _ := evalAIReady(res)
	setLanguage(ad, lang)
	out["ai_ready"] = MetricResult{Data: ad, Score: Score(a)}

	return out
}
