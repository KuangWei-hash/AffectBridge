# EVALUATION — AffectBridge Experimental Plan

## Status

**Experimental evaluation plan — v0.1**

The central purpose of AffectBridge is not to argue from architecture alone.

The method should be judged by whether it produces better character behavior in practice.

---

## 1. Primary hypothesis

AffectBridge evaluates the following hypothesis:

> **Explicit structured appraisal plus persistent affective state can produce NPC emotional responses that players judge as more believable, character-consistent, appropriately intense, and temporally coherent than a carefully engineered prompt-only baseline.**

The experiment must allow this hypothesis to fail.

---

## 2. What is NOT being claimed

AffectBridge does not need to prove:

```text
The NPC experiences real emotion.

The architecture reproduces human cognition.

ALMA is a complete theory of human psychology.

LLM appraisal is psychologically identical to human appraisal.
```

The target is practical game-character behavior.

The relevant question is:

> Does the resulting NPC react better from the player's point of view?

---

## 3. Primary comparison

At minimum, compare two systems.

### System A — Strong Full-Prompt Baseline

```text
Character description
+
Relevant memory / dialogue history
+
Current event
+
Emotion / behavior instructions
        ↓
LLM
        ↓
NPC response
```

This baseline must be treated seriously.

It should be carefully engineered rather than intentionally weakened.

### System B — AffectBridge

```text
Story
+
Character-specific viewpoints
        ↓
input-appraisal tags 轉換系統
        ↓
18 Appraisal Tags
+
Semantic intensity
        ↓
Numeric conversion
        ↓
ALMA Emotion / Mood
        ↓
Affect Snapshot
        ↓
LLM
        ↓
NPC response
```

Where practical, both systems should use the same base LLM.

---

## 4. Why the baseline must be strong

A weak baseline would only prove:

> A bad prompt loses to a structured system.

That is not interesting.

The intended question is:

> **Does explicit affective structure provide measurable value even when compared with a well-designed prompt-only NPC?**

If possible, the full-prompt baseline should be optimized independently or reviewed by people who believe prompt engineering is sufficient.

This reduces the criticism that AffectBridge wins only because the competing prompt was poorly designed.

---

## 5. Primary human-evaluation dimensions

### Emotional appropriateness

> Does the emotional response fit what happened?

### Character consistency

> Does the reaction fit this character's established priorities, loyalties, sensitivities, and values?

### Intensity appropriateness

> Is the strength of the reaction appropriate to the event from this character's perspective?

### Temporal emotional continuity

> Do important earlier events continue to influence the character in a believable way?

### Affective consequence

> Does betrayal, rescue, humiliation, promise-breaking, reconciliation, or similar events leave meaningful consequences?

### Overall believability

> Does the NPC feel like a coherent continuing character rather than a sequence of individually plausible responses?

---

## 6. Recommended blind A/B design

Players should not know which architecture produced each NPC.

Example:

```text
NPC A
NPC B
```

Both appear to be the same fictional character.

The participant experiences equivalent scenarios and then rates the outputs.

Questions may include:

```text
Which NPC felt more emotionally consistent?

Which NPC reacted more appropriately to the severity of events?

Which NPC better preserved the character's values?

Which NPC felt more affected by important previous events?

Which NPC felt more believable overall?
```

---

## 7. Longitudinal scenarios are essential

AffectBridge should not be evaluated only on single-turn emotional reactions.

A single LLM response can already be highly convincing.

The more important test is whether the character remains emotionally coherent across a sequence.

Example scenario:

```text
1. Player is initially a stranger.
2. Player helps the NPC.
3. Trust gradually develops.
4. Player makes an explicit promise.
5. Player breaks the promise.
6. Player apologizes.
7. Player breaks trust again.
8. Player later takes a serious risk to save the NPC.
9. Player asks for forgiveness.
```

The evaluation should examine whether the character's later reactions make sense in light of the whole affective history.

---

## 8. Character-sensitive testing

A structured appraisal system should also produce different reactions for characters with different priorities.

Example:

```text
Event:
The player breaks a promise.

Character A:
Treats promises as extremely important.

Character B:
Is unusually forgiving about broken promises.
```

Expected system property:

```text
Same event
+
Different character-specific viewpoints
        ↓
Different appraisal intensity
        ↓
Different affective trajectory
```

This should be measurable.

---

## 9. Appraisal-level evaluation

The system can also be evaluated before final dialogue generation.

For a curated set of stories, human annotators can judge whether:

```text
GoodEvent = TRUE/FALSE
BadEvent = TRUE/FALSE
BadActOther = TRUE/FALSE
...
```

and whether the semantic intensity is reasonable.

This allows separate measurement of:

```text
Story → Appraisal quality
```

from:

```text
Appraisal → NPC response quality
```

That separation is valuable because it helps locate where improvements come from.

---

## 10. Repeatability testing

The same scenario should be run multiple times.

Measure:

- tag stability;
- intensity stability;
- final-response variance;
- character-consistency variance.

A system that occasionally produces an excellent reaction but frequently drifts may be less useful than one that is consistently good.

---

## 11. Engineering-cost measurements

AffectBridge deliberately adds structure and model calls.

Therefore quality should be evaluated together with cost.

Track at least:

```text
LLM calls per event
Input tokens
Output tokens
Latency
Estimated inference cost
CPU / GPU requirements where relevant
```

The final result should ideally describe a trade-off:

```text
Additional inference cost
        ↕
Improvement in NPC affective quality
```

This makes the work useful for real game-development decisions.

---

## 12. Example rating instrument

A 7-point Likert scale is suitable for an initial study:

```text
1 = strongly disagree
7 = strongly agree
```

Possible statements:

```text
The NPC's emotional reaction was appropriate.

The NPC behaved consistently with its established character.

The strength of the NPC's reaction matched the importance of the event.

Earlier events continued to influence the NPC naturally.

The NPC's emotional state changed in a believable way over time.

The NPC felt like a coherent continuing character.

The NPC felt believable as a game character.
```

The final questionnaire should be reviewed before formal publication.

---

## 13. Steam game as public experiment

A public playable game can act as both:

```text
Game
+
Technology demonstration
+
Experimental platform
```

A useful release design could include:

```text
Normal Play Mode
```

for ordinary players, and optionally:

```text
Research / A-B Mode
```

for controlled comparison.

The game should still be enjoyable as a game.

The experimental architecture should not require players to understand ALMA in advance.

The strongest demonstration is when players notice the behavioral difference before they know how the system works.

---

## 14. Interpretation of possible results

### AffectBridge performs better

This supports the claim that explicit structured appraisal and persistent affective dynamics add practical value beyond prompt engineering alone.

### No meaningful difference

The extra architecture may not justify its cost for the tested scenarios.

### Full-Prompt baseline performs better

This is also useful evidence.

It may indicate that the structured system needs revision, that the tested scenarios do not require persistent affect, or that modern LLM context is sufficient for the measured task.

AffectBridge should report the result honestly.

---

## 15. Success criterion

The strongest possible result is not:

> AffectBridge sounds more emotional.

It is:

> **Across controlled multi-event interactions, human evaluators consistently judge AffectBridge characters as more emotionally coherent, character-consistent, and appropriately reactive than a strong prompt-only baseline.**

That is the result this project should attempt to demonstrate.
