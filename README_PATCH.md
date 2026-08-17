# Suggested README additions for AffectBridge

These blocks are intended to be integrated into the existing README rather than replacing it.

---

## Add after `# Proposed Runtime Flow`

### Structured Appraisal Front-End

The initial AffectBridge implementation uses an LLM-based structured appraisal front-end before the original ALMA runtime.

Rather than requiring a manually authored appraisal rule for every possible event, AffectBridge experiments with:

```text
Narrative Story
+
Character-specific viewpoints
        ↓
LLM-based appraisal inference
        ↓
18 ALMA Appraisal Tags
+
Semantic intensity
        ↓
ALMA-compatible numeric appraisal
        ↓
Original ALMA runtime
```

The LLM-based conversion layer is referred to in this project as the **input-appraisal tags 轉換系統**.

Each ALMA Appraisal Tag is expressed as an explicit natural-language question. Specialized workers determine whether the tag applies and, when it does, estimate its semantic intensity.

The LLM does not directly choose arbitrary appraisal floating-point values. Instead, intensity is classified into semantic levels and converted deterministically by program logic.

This makes the appraisal stage easier to inspect, test, calibrate, and compare across models.

See:

- [`docs/METHOD.md`](docs/METHOD.md)
- [`docs/APPRAISAL_TAGS.md`](docs/APPRAISAL_TAGS.md)
- [`docs/EVALUATION.md`](docs/EVALUATION.md)

---

## Add near `# Core Architectural Hypothesis`

A second experimental hypothesis is:

> **A structured appraisal layer may provide more stable and character-specific affective interpretation than asking a single prompt to implicitly maintain the entire appraisal and emotional process.**

This claim is not assumed to be true.

AffectBridge is being built specifically so it can be tested against a strong prompt-only baseline.

---

## Add near the project status / roadmap

### Current Research Direction

The current experimental sequence is:

```text
Story
        ↓
Character-specific appraisal
        ↓
18 ALMA Appraisal Tags
        ↓
Semantic intensity
        ↓
ALMA Emotion / Mood
        ↓
LLM expression
```

The first public evaluation is intended to compare this architecture with a carefully engineered full-prompt character using the same base LLM where practical.
