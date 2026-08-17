# METHOD — Structured LLM Appraisal Front-End for AffectBridge

## Status

**Experimental method specification — v0.1**

Author: **KuangWei Chen**
Project: **AffectBridge**
Year: **2026**

This document records the current AffectBridge method for converting narrative game events into structured appraisal input for an ALMA-based affect backend.

The method is experimental. Its purpose is to be implemented, tested, compared against strong prompt-only baselines, and revised according to evidence.

---

## 1. Objective

AffectBridge is not attempting to reproduce human psychology exactly.

The practical objective is:

> **Build NPCs whose emotional responses are more believable, character-consistent, and temporally coherent during sustained interaction.**

Modern LLMs are already strong at dialogue generation and role performance. The architectural question explored here is whether persistent affect should remain entirely implicit inside prompt/context, or whether part of it should be represented as explicit structured state.

AffectBridge takes the latter approach.

---

## 2. Core architectural hypothesis

AffectBridge is built around the hypothesis that:

> **Language generation and persistent character state are different engineering problems.**

The LLM performs semantic interpretation and expression.

The affect model maintains explicit affective continuity.

Conceptually:

```text
The LLM performs the character.
The affect model gives the character continuity.
```

---

## 3. Relationship to ALMA

AffectBridge uses the original ALMA runtime as an external affect backend.

ALMA is not an invention of AffectBridge.

The original ALMA architecture uses subjective appraisal rules to transform relevant symbolic input into appraisal-related conditions that drive emotion and mood processing.

AffectBridge changes one important part of that pipeline.

### Original direction

```text
Game / World Event
        ↓
Manually authored Appraisal Rules
        ↓
Appraisal / EEC representation
        ↓
ALMA Emotion
        ↓
ALMA Mood
```

### AffectBridge experimental direction

```text
Narrative Story
+
Character-specific viewpoints
        ↓
LLM-based structured appraisal inference
        ↓
18 ALMA Appraisal Tags
+
Semantic intensity
        ↓
Deterministic numeric conversion
        ↓
Original ALMA runtime
        ↓
Emotion / Mood
        ↓
LLM expression
```

Therefore, AffectBridge does **not** discard ALMA's appraisal representation or downstream affect processing.

It experiments with replacing the manually authored **event-to-appraisal rule layer** with an LLM-based front-end.

---

## 4. The input-appraisal tags 轉換系統

The LLM-based front-end is called:

> **input-appraisal tags 轉換系統**

Its job is deliberately narrow.

It does **not** directly decide the final emotion.

Its responsibility is:

```text
Story
+
Character-specific viewpoints
        ↓
Appraisal Tags
+
Appraisal intensity
```

ALMA remains responsible for downstream emotion and mood dynamics.

---

## 5. Story as the event representation

Raw dialogue is not directly treated as appraisal state.

Recent dialogue and game events are first represented as a compact **Story**.

Initial implementation direction:

```text
Recent player / NPC dialogue
+
Relevant game events
        ↓
Story
```

The implemented v0.1 first stage uses the latest ten chronological utterances
(player and character messages each count as one) and requests a maximum
128-token Story. Every Story clause must identify its subject explicitly using
`玩家` or a known character name; personal pronouns are rejected. Its code
contract, safety boundary, and integration lifecycle
are documented in [`STORY_PIPELINE.md`](STORY_PIPELINE.md).

The Story should describe what happened without pre-labeling the emotion.

For example, the Story should prefer:

```text
The player promised Lisa that he would return before nightfall,
but he did not return.
```

over:

```text
Lisa was betrayed and became angry.
```

The appraisal system should infer the meaning rather than receive the emotional conclusion in advance.

---

## 6. Character-specific viewpoints

AffectBridge does not attempt to encode every ordinary human preference or every possible event rule.

Instead, the character prompt describes only the ways in which the character differs meaningfully from an ordinary common-sense baseline.

Conceptually:

```text
Character Appraisal
=
Common-sense baseline
+
Character-specific deviations / priorities
```

Example:

```text
Lisa does not easily trust strangers.

Lisa treats explicit promises more seriously than most people.

Lisa reacts especially strongly to betrayal from someone she has accepted
as one of her own.
```

This is intentionally different from a large hand-authored appraisal-rule database.

The LLM supplies ordinary semantic common sense.

The character-specific viewpoints supply the character's unusual priorities, sensitivities, loyalties, dislikes, and value deviations.

---

## 7. Appraisal Tags become explicit natural-language questions

AffectBridge does not rely on the model understanding technical tag names in isolation.

Each of the 18 ALMA Appraisal Tags is rewritten as an explicit natural-language TRUE/FALSE question.

Example:

```text
GoodEvent
```

becomes:

```text
故事中是否發生了對 Lisa 本人有利、符合 Lisa 的利益或目標的事情？
```

Likewise:

```text
BadActOther
```

becomes:

```text
故事中是否有其他人做了某個依照 Lisa 的價值標準來看
錯誤、應受責備或不被認同的行為？
```

The canonical v0.1 question set is maintained separately in:

```text
docs/APPRAISAL_TAGS.md
```

---

## 8. Shared ontology, specialized workers

The current design uses specialized appraisal workers.

Each worker:

- receives the same character;
- receives the same character-specific viewpoints;
- receives the same Story;
- knows the full 18-question appraisal ontology;
- is assigned exactly one target question;
- answers only that target question.

Example:

```text
                    Story
                      │
        ┌─────────────┼─────────────┐
        ↓             ↓             ↓
   Worker 01      Worker 02      ... Worker 18
   GoodEvent      BadEvent           NastyThing
```

The full ontology gives each worker semantic boundaries between nearby concepts.

The single-target restriction reduces output ambiguity.

---

## 9. Stage 1 — Boolean appraisal decision

Every worker first answers:

```text
是/否: 是 或 否
理由: 一句簡短理由
```

Important rules:

- Only evaluate the assigned appraisal question.
- Use only the Story and character information.
- If evidence is insufficient, answer `否`.
- Do not invent events, relationships, expectations, intentions, or internal thoughts that are not supported.
- If the character has no special viewpoint about something, ordinary reasonable common sense may be used.
- If the character's explicit viewpoint conflicts with ordinary common sense, the character's viewpoint takes priority.

This first stage determines whether the appraisal tag is active.

---

## 10. Stage 2 — Semantic intensity

Only if the Boolean result is `是`, the same worker continues to estimate intensity.

The LLM does **not** directly output a floating-point appraisal value.

Instead, it selects one semantic level:

```text
極輕微
輕微
中等
強烈
非常強烈
極端
```

If the Boolean result is `否`:

```text
強度: 不適用
```

This separation is intentional:

```text
Tag existence
        ↓
Semantic strength
        ↓
Numeric calibration
```

---

## 11. Why semantic intensity instead of arbitrary numbers?

Asking an LLM to directly produce values such as:

```text
0.63
-0.81
0.47
```

mixes semantic reasoning with numeric calibration.

AffectBridge separates them.

The LLM answers the question it is naturally suited for:

> How important or intense is this event from this character's point of view?

Program logic then performs deterministic numeric conversion.

Advantages:

- easier to inspect;
- easier to compare across models;
- easier to reproduce;
- easier to calibrate;
- easier to adjust without rewriting appraisal prompts;
- less dependence on arbitrary decimal choices made by the LLM.

---

## 12. Numeric conversion

ALMA-compatible appraisal values ultimately require numeric magnitudes.

AffectBridge therefore maps semantic intensity to numeric magnitude outside the LLM.

Example calibration only:

```text
極輕微       → 0.10
輕微         → 0.25
中等         → 0.45
強烈         → 0.65
非常強烈     → 0.85
極端         → 1.00
```

These numbers are **not claimed to be psychologically canonical**.

They are calibration parameters.

The appraisal tag semantics determine direction/sign.

For example:

```text
GoodEvent + 強烈
→ positive desirability magnitude

BadEvent + 強烈
→ negative desirability magnitude
```

The numeric mapping should ultimately be validated experimentally.

---

## 13. Tag-specific intensity meaning

Intensity should not be interpreted identically for all tags.

Examples:

### GoodEvent

```text
How strongly does the event benefit the character's interests or goals?
```

### BadEvent

```text
How strongly does the event harm the character's interests or goals?
```

### GoodActOther

```text
How strongly does the character approve of or praise the other person's action?
```

### BadActOther

```text
How strongly does the character condemn or blame the other person's action?
```

### NiceThing

```text
How strongly does the character like, appreciate, or feel attracted to the object/person?
```

### NastyThing

```text
How strongly does the character dislike or reject the object/person?
```

Therefore, each worker may share the same output scale while using a tag-specific interpretation of what "intensity" means.

---

## 14. Intended prompt boundary

Recommended first implementation:

### System Prompt

Contains stable worker behavior:

```text
Character identity
Character-specific viewpoints
18-question appraisal ontology
Worker's target question
Boolean judgment rules
Intensity rules
Output schema
```

### User Prompt

Contains dynamic input:

```text
Story
```

This keeps worker semantics stable while allowing the Story to change every call.

---

## 15. Failure localization

An important engineering purpose of this architecture is inspectability.

If an NPC response is poor, AffectBridge can ask:

```text
Was the Story wrong?

Was the Appraisal Tag wrong?

Was the intensity wrong?

Was numeric calibration wrong?

Did ALMA produce an unsuitable emotion or mood transition?

Did the final LLM expression fail?
```

A pure-prompt architecture tends to collapse these stages into one generation step.

AffectBridge intentionally keeps them separable.

---

## 16. Current project-specific contributions

The following items describe the current AffectBridge implementation direction.

They are **not** claims that no related idea exists in prior research.

1. Using an LLM as the semantic bridge between narrative game events and ALMA appraisal input.
2. Rewriting the 18 ALMA Appraisal Tags as explicit natural-language Boolean questions.
3. Representing character individuality primarily through character-specific appraisal deviations and priorities rather than exhaustive hand-authored event rules.
4. Using specialized workers to evaluate individual appraisal dimensions.
5. Separating appraisal-tag activation from appraisal intensity.
6. Estimating intensity with semantic ordinal labels rather than free-form floating-point values.
7. Converting semantic intensity deterministically into ALMA-compatible numeric values.
8. Retaining explicit ALMA emotion and mood dynamics instead of asking the LLM to implicitly reconstruct persistent affect every turn.
9. Using the LLM again after affect processing as the character's expression layer.

---

## 17. Main empirical question

The method should ultimately be judged by player experience.

The primary research question is:

> **Does structured LLM appraisal combined with explicit ALMA affect dynamics produce NPC emotional behavior that players judge as more believable, character-consistent, appropriately intense, and temporally coherent than a carefully engineered prompt-only baseline?**

AffectBridge does not assume the answer.

The purpose of the project is to test it.

---

## 18. Research lineage

AffectBridge builds on prior work including:

- Patrick Gebhard, **ALMA — A Layered Model of Affect**, AAMAS 2005.
- Work exploring LLM-based appraisal and affective reasoning.
- Appraisal-theory-based approaches to computational emotion.
- Modern LLM-driven character systems.

AffectBridge should clearly distinguish:

```text
Prior affective-computing theory
from
AffectBridge-specific engineering method
from
Experimental evidence produced by this project
```

That distinction should remain explicit in future documentation and publications.
