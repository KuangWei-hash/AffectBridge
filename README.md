# AffectBridge

> **Bringing classical computational affect models into modern LLM-driven game characters.**

AffectBridge is an independent open-source experiment exploring how **structured affective state** can be integrated with modern Large Language Models to build game characters with more persistent, coherent, inspectable, and explainable emotional behavior.

The first implementation uses **ALMA — A Layered Model of Affect**, created by **Patrick Gebhard at the German Research Center for Artificial Intelligence (DFKI)**, as its initial affective backend.

AffectBridge does **not** claim to have invented ALMA, layered affect modeling, or the psychological theories behind it.

Quite the opposite:

> **This project exists because I believe ALMA represents an excellent architectural direction for artificial characters, and that its ideas are worth revisiting, using, testing, and extending in the era of modern LLMs.**

The initial goal is deliberately modest:

> **Use the original ALMA runtime without modifying or redistributing it, connect it to a modern LLM, build a playable character, and observe what structured persistent affect contributes in practice.**

---

# Why This Project Exists

Modern LLMs have made it surprisingly easy to create characters that can talk.

A typical implementation may place almost everything into prompts and context:

```text
Personality
Emotion
Relationship
Memory
Goals
World State
Behavior Rules
Conversation History
        │
        ▼
      Prompt
        │
        ▼
       LLM
        │
        ▼
    Character
```

This approach can work extremely well.

AffectBridge is not intended to argue that prompts are useless.

Instead, it asks a narrower architectural question:

> **Should the prompt and LLM context be responsible for maintaining the entire internal psychological state of a persistent character?**

AffectBridge explores an alternative:

```text
                    Game / Player
                         │
                         ▼
                    Perception
                         │
                         ▼
                     Appraisal
                         │
                         ▼
                ┌─────────────────┐
                │  Affect Model   │
                │                 │
                │  Personality    │
                │  Mood           │
                │  Emotion        │
                └────────┬────────┘
                         │
                         ▼
                 Persistent State
                         │
                         ▼
                       LLM
                         │
                         ▼
              Dialogue / Intent / Action
```

The central idea is:

> **The LLM performs the character.
> The affect model gives the character continuity.**

---

# Our Position on ALMA

AffectBridge does **not** claim to have invented ALMA.

It does not claim to have invented:

* layered affect modeling
* personality modeling
* cognitive appraisal
* OCC emotions
* PAD mood representation
* emotion decay
* computational affect

These ideas come from decades of prior research.

The first affective backend used by AffectBridge is based specifically on the work of **Patrick Gebhard and DFKI**.

AffectBridge exists because I believe that ALMA's basic architectural direction remains highly relevant.

ALMA made an important distinction between affective states operating at different time scales:

```text
Long-term
┌────────────────────┐
│    Personality     │
└─────────┬──────────┘
          │
          ▼
Medium-term
┌────────────────────┐
│        Mood        │
└─────────┬──────────┘
          │
          ▼
Short-term
┌────────────────────┐
│      Emotion       │
└────────────────────┘
```

That separation is particularly interesting today.

Modern generative characters often attempt to represent personality, mood, transient emotion, relationship history, and memory together through natural-language context.

ALMA took a very different approach:

> **Internal affect was explicit state.**

AffectBridge asks what happens when that older computational-affect philosophy is connected to modern language models.

The purpose is not to replace or obscure ALMA.

The purpose is to:

* clearly acknowledge the original ALMA research
* experiment with ALMA in modern LLM-driven characters
* make integration easier
* evaluate where the model works well
* identify where the model does not work well
* explore extensions suitable for modern game characters
* bring renewed attention to valuable computational-affect research
* build on the knowledge that earlier researchers made publicly available

In short:

> **ALMA is not our invention.
> We think ALMA is a valuable idea worth carrying forward.**

---

# About ALMA

**ALMA — A Layered Model of Affect** was introduced by Patrick Gebhard in:

> **Patrick Gebhard**
> *ALMA — A Layered Model of Affect*
> Fourth International Joint Conference on Autonomous Agents and Multiagent Systems
> AAMAS 2005

Official ALMA website:

https://alma.dfki.de/

Original ALMA source repository:

https://github.com/A-L-M-A/ALMA

DFKI:

https://www.dfki.de/

ALMA combines ideas from established psychological and affective-computing frameworks, including:

* Big Five personality traits
* cognitive appraisal theory
* OCC emotion modeling
* PAD dimensional mood representation
* emotion intensity
* emotion decay
* mood dynamics
* personality-dependent affective baseline

AffectBridge does not claim that ALMA is a biologically exact model of human psychology or neuroscience.

It is treated here as a computational affect model for artificial characters.

---

# Initial Experiment: Original ALMA + Modern LLM

The first version of AffectBridge intentionally avoids reimplementing ALMA.

Instead:

```text
AffectBridge
Apache-2.0
     │
     │ integration / protocol
     ▼
Original ALMA Java Runtime
installed separately by user
     │
     ▼
Emotion / Mood / Personality
     │
     ▼
LLM
     │
     ▼
Playable Character
```

The purpose of this decision is simple:

> **Before designing a new affect system, first determine how useful the existing ALMA model actually is when combined with a modern LLM.**

This provides a real baseline.

If ALMA works well, that is useful evidence.

If ALMA exposes limitations, those limitations become useful information for future work.

---

# ALMA Is an External Dependency

The original ALMA software is **not included in this repository**.

AffectBridge does not redistribute:

* ALMA source code
* ALMA Java binaries
* ALMA JAR files
* ALMA configuration files
* ALMA documentation
* ALMA assets
* other original ALMA materials

Users who want to use the ALMA backend must obtain ALMA themselves from its official source and comply with its original license.

Conceptually:

```text
User
 │
 ├── downloads AffectBridge
 │       └── Apache-2.0
 │
 └── separately obtains ALMA
         └── original ALMA license
```

A future setup may look similar to:

```bash
export ALMA_HOME=/path/to/alma
```

AffectBridge then communicates with the locally installed ALMA runtime.

---

# Licensing Boundary

AffectBridge and ALMA are separate projects with separate licenses.

```text
AffectBridge
    │
    └── Apache License 2.0

ALMA
    │
    └── ALMA's original license
```

The Apache License 2.0 applies only to code and material authored specifically for AffectBridge unless otherwise stated.

It does **not** apply to ALMA.

AffectBridge does not attempt to relicense ALMA.

Users are responsible for reviewing and complying with the ALMA license separately.

---

# Why Not Reimplement ALMA Immediately?

A clean-room reimplementation may eventually make sense.

But that is not the first question this project needs to answer.

The first question is:

> **Does ALMA actually improve the experience of a modern LLM-driven game character?**

Reimplementing the entire model before answering that question would add unnecessary engineering work and potentially introduce implementation differences.

Using the original runtime provides a useful reference point:

```text
Original Research
       │
       ▼
Original Runtime
       │
       ▼
Modern Integration
       │
       ▼
Actual Player Experience
```

Only after evaluating this baseline does it make sense to ask whether ALMA should be:

* reimplemented
* simplified
* modernized
* extended
* replaced
* partially learned
* neuralized
* combined with another affect model

---

# Core Architectural Hypothesis

AffectBridge explores the hypothesis that:

> **Language generation and persistent character state are different engineering problems.**

Large language models are extremely capable at:

* language
* reasoning
* improvisation
* semantic interpretation
* dialogue generation
* world knowledge

But persistent character state may benefit from explicit representation.

Instead of requiring the LLM to reconstruct everything from conversation text:

```text
Conversation History
        │
        ▼
       LLM
        │
        ▼
"What should I feel?"
```

AffectBridge explores:

```text
Character State
        │
        ▼
       LLM
        │
        ▼
"How should I express this state?"
```

---

# Personality Is Not Mood

AffectBridge adopts ALMA's important distinction between personality and temporary affect.

For example:

```text
Personality

Agreeableness = high
Neuroticism   = low
```

does not prevent:

```text
Current Mood

Hostile
```

after a sufficiently negative event.

A friendly person can become angry.

An anxious person can temporarily become joyful.

A hostile mood does not necessarily indicate a hostile personality.

These states operate at different time scales.

---

# Mood Is Not Emotion

Emotion is relatively short-lived.

Mood persists longer.

Conceptually:

```text
Player insults character
        │
        ▼
Anger = 0.90
        │
        ▼
Anger = 0.62
        │
        ▼
Anger = 0.27
        │
        ▼
Anger ≈ 0
```

But the accumulated affect may have shifted the character's mood.

Therefore:

```text
Emotion disappears
        ≠
Character instantly returns to previous psychological state
```

This kind of temporal behavior is one of the things AffectBridge intends to evaluate.

---

# Proposed Runtime Flow

The first experimental pipeline is:

```text
Player Input / Game Event
          │
          ▼
┌──────────────────────────┐
│      Semantic Layer      │
│                          │
│          LLM             │
└────────────┬─────────────┘
             │
             ▼
     Structured Appraisal
             │
             ▼
┌──────────────────────────┐
│       Original ALMA      │
│                          │
│ Personality              │
│ Emotion                  │
│ Mood                     │
│ Affect Dynamics          │
└────────────┬─────────────┘
             │
             ▼
       Affect Snapshot
             │
             ▼
┌──────────────────────────┐
│           LLM            │
│                          │
│ Dialogue / Expression    │
└────────────┬─────────────┘
             │
             ▼
       Character Output
```

The LLM may therefore play two different roles.

### Semantic interpretation

Determine what an event means in context.

### Expression

Turn a structured internal state into natural character behavior and dialogue.

The affect model remains responsible for persistent affective state.

---

# Example

Suppose a player previously promised:

> "I'll stay with you during the battle."

But later runs away.

The semantic layer may interpret the event:

```json
{
  "agency": "player",
  "desirability": -0.87,
  "unexpectedness": 0.68,
  "blameworthiness": 0.92
}
```

The original ALMA runtime processes the appraisal and produces affective changes.

Conceptually:

```text
Anger       ↑
Distress    ↑
Reproach    ↑

Pleasure    ↓
Arousal     ↑
```

A resulting affect snapshot might look like:

```json
{
  "mood": {
    "pleasure": -0.48,
    "arousal": 0.61,
    "dominance": 0.23
  },
  "emotions": {
    "anger": 0.78,
    "distress": 0.42,
    "reproach": 0.71
  }
}
```

The LLM is then asked to express that state.

It is not asked to invent the state from nothing.

---

# First Playable Character

The first public experiment should intentionally remain small.

A single character is enough.

The player should be able to:

* talk to the character
* praise them
* insult them
* make promises
* break promises
* apologize
* help them
* threaten them
* wait for emotions to decay
* observe mood changes
* reset the LLM context
* continue interacting with the same affective state

During development, internal state should remain visible.

Example debug panel:

```text
CURRENT MOOD

Pleasure      -0.42
Arousal       +0.63
Dominance     +0.18


ACTIVE EMOTIONS

Anger          0.76
Distress       0.38
Reproach       0.64
Hope           0.09
```

The system is intended to be inspectable.

Not magical.

---

# Experimental Questions

The first version of AffectBridge aims to answer several practical questions.

## Does ALMA Feel Useful?

Does the affective state produce behavior that players actually perceive as more coherent?

## Does Emotion Evolve Naturally?

Does emotion decay produce convincing character behavior?

## Does Mood Matter?

Can players perceive a difference between short-lived emotions and accumulated mood?

## Does Personality Remain Stable?

Can a character experience radically different emotions without losing its long-term identity?

## Does State Survive Context Reset?

If the LLM conversation context is cleared while affect state remains:

> Does the character still feel psychologically continuous?

## Does the LLM Actually Respect Affect State?

Given:

```text
Anger = 0.80
```

does the language model reliably express anger?

Given later:

```text
Anger = 0.05
```

does it calm down appropriately?

---

# Future Comparison: Prompt Character vs Structured Character

A later phase may compare:

```text
Prompt Character

Personality
Emotion
Memory
Relationship
History
      │
      ▼
    Prompt
      │
      ▼
     LLM
```

against:

```text
Structured Character

Persistent State
      │
      ├── Personality
      ├── Mood
      ├── Emotion
      ├── Memory
      └── Relationship
      │
      ▼
     LLM
```

The purpose is not to assume one approach is superior.

The purpose is to measure it.

---

# Long-Term Research Question

One particularly interesting future question is:

> **Can structured character state reduce the amount of language-model scale required for believable long-running characters?**

A future experiment may compare:

```text
Large LLM
+
Prompt Persona
+
Large Context
+
RAG
```

against:

```text
Small LLM
+
Structured Psyche
+
RAG
```

Possible evaluation dimensions include:

* personality stability
* emotional continuity
* emotional causality
* emotional decay
* long-horizon consistency
* contradiction resistance
* context-reset robustness
* model-swap robustness
* token consumption
* inference cost

A possible hypothesis is:

> **Character fidelity is not purely a model-scale problem.**

This is a hypothesis.

Not a claim.

AffectBridge exists to generate evidence.

---

# Possible Future Architecture

If the ALMA experiment proves useful, AffectBridge may evolve toward:

```text
                    Game World
                        │
                        ▼
                   Perception
                        │
                        ▼
                Semantic Appraisal
                        │
                        ▼
              ┌───────────────────┐
              │  Character Psyche │
              │                   │
              │ Personality       │
              │ Mood              │
              │ Emotion           │
              │ Relationship      │
              │ Memory            │
              │ Goals             │
              │ Beliefs           │
              └─────────┬─────────┘
                        │
                        ▼
                       LLM
                        │
             ┌──────────┴──────────┐
             ▼                     ▼
          Dialogue               Intent
                                    │
                                    ▼
                              Game Behavior
```

ALMA would provide part of the foundation rather than necessarily defining the final architecture.

---

# Possible Future Work

Future experiments may include:

* RAG
* episodic memory
* semantic memory
* relationship modeling
* trust
* affection
* respect
* fear
* resentment
* goals
* beliefs
* LLM-based appraisal
* learned appraisal
* neural affect models
* learned emotion dynamics
* learned emotion decay
* ALMA-compatible independent runtime
* Godot integration
* Unity integration
* Unreal integration
* multiplayer NPC state
* deterministic replay
* simulation tools
* visualization tools
* character-state debugging
* automated benchmarks
* human blind testing

None of these are required for the initial experiment.

---

# What AffectBridge Does Not Claim

AffectBridge does **not** claim:

* to have invented ALMA
* to have invented layered affect
* to have invented computational emotion
* to have invented cognitive appraisal
* to have invented OCC
* to have invented PAD
* to perfectly simulate human psychology
* to simulate human neuroscience
* to create consciousness
* that prompts are useless
* that prompt-driven characters are inherently wrong
* that small models universally outperform large models
* that ALMA is the only valid affect architecture
* that AffectBridge is an official continuation of ALMA

The project is an independent engineering experiment.

---

# What AffectBridge Does Claim

Very little — intentionally.

The current claim is simply:

> **ALMA is an interesting and historically important computational-affect architecture that deserves to be tested with modern language models.**

Everything beyond that should be demonstrated through implementation and experiments.

---

# Development Principles

AffectBridge should remain:

## Inspectable

Important internal state should be visible during development.

## Reproducible

Experiments should be repeatable whenever practical.

## Modular

The LLM and affect model should remain separable components.

## Provider Independent

Changing LLM provider should not require replacing the entire architecture.

## Game-Engine Independent

The core bridge should not depend on one game engine.

## Affect-Model Friendly

ALMA is the first backend.

It does not need to be the last.

---

# Repository Structure

A possible early repository structure:

```text
affect-bridge/
│
├── README.md
├── LICENSE
├── NOTICE
│
├── docs/
│   ├── architecture.md
│   ├── alma-setup.md
│   ├── experiments.md
│   └── references.md
│
├── src/
│   ├── bridge/
│   ├── protocol/
│   ├── llm/
│   └── character/
│
├── examples/
│   └── minimal-character/
│
└── tests/
```

The structure will evolve during experimentation.

---

# Setup

AffectBridge does not provide ALMA.

Users must obtain the official ALMA runtime separately.

Official project:

https://alma.dfki.de/

Original source repository:

https://github.com/A-L-M-A/ALMA

After installing or compiling ALMA, AffectBridge will eventually support configuration similar to:

```bash
export ALMA_HOME=/path/to/alma
```

Detailed setup instructions will be added once the first bridge implementation is available.

---

# License

AffectBridge is licensed under the:

## Apache License 2.0

See the `LICENSE` file for the complete license terms.

This license applies only to AffectBridge itself.

It does **not** apply to ALMA or any other external dependency.

---

# A Note on Sharing Improvements

AffectBridge is intentionally licensed permissively.

You are welcome to:

* use it
* study it
* modify it
* fork it
* redistribute it
* integrate it into games
* integrate it into proprietary software
* use it commercially
* build products around it
* make money with it

You are not required by the license to publish your modifications.

However, if you discover:

* a better affect model
* a better parameterization
* an interesting extension
* a bug in the model
* a better appraisal method
* a more realistic emotional dynamic
* a performance improvement
* a useful experimental result

I sincerely hope you will consider sharing what you learned with everyone.

Not because the license forces you to.

Because this project itself exists only because researchers before us chose to publish what they discovered.

Patrick Gebhard and other researchers published their work.

That allowed people years later to study it, question it, build on it, and bring it into entirely new technological contexts.

That is how knowledge survives.

So:

> **Use it however you want.**
> **Build whatever you want.**
> **Profit from it if you want.**
> **If you make it better, please consider sharing what you learned.**

The goal is not to prevent anyone from benefiting from knowledge.

The hope is simply that useful knowledge continues moving forward.

---

# Attribution and Acknowledgements

AffectBridge exists because of previous research in affective computing, intelligent virtual agents, psychology, and computational emotion.

Special acknowledgement goes to:

**Patrick Gebhard**

and the researchers at:

**German Research Center for Artificial Intelligence (DFKI)**

for developing and publishing ALMA.

Primary reference:

> Patrick Gebhard
> **ALMA — A Layered Model of Affect**
> AAMAS 2005

Official project:

https://alma.dfki.de/

AffectBridge intends to clearly preserve the origin and academic lineage of the ideas it builds upon.

If this project helps bring renewed attention to ALMA and related computational-affect research, that is considered a success.

---

# Relationship With the Original Authors

AffectBridge is currently an **independent project**.

It is not affiliated with, sponsored by, maintained by, or endorsed by Patrick Gebhard, DFKI, or the ALMA project.

If the project reaches a useful playable state, I intend to make the original ALMA researchers aware of the experiment and welcome technical feedback, corrections, historical context, or suggestions.

The purpose is cooperation and continuation of knowledge — not appropriation of prior work.

---

# Disclaimer

AffectBridge is experimental research software.

It should not be interpreted as:

* a psychological diagnostic tool
* a medical system
* an accurate model of human cognition
* an accurate model of the human brain

Use it as an experimental computational model for artificial characters.

---

# Contributing

Contributions are welcome.

Useful contributions may include:

* bridge implementation
* ALMA compatibility testing
* documentation
* reproducible experiments
* test scenarios
* UI/debugging tools
* LLM-provider adapters
* benchmark designs
* game-engine adapters
* performance improvements
* failure reports
* academic references
* corrections to interpretations of ALMA

Contradictory evidence is welcome too.

If an experiment demonstrates that an assumption behind AffectBridge is wrong, that is a useful result.

The goal is not to prove a predetermined conclusion.

The goal is to find out what actually works.

---

# Status

**Experimental — early development**

Current intended milestone:

```text
Original ALMA
      +
Minimal AffectBridge
      +
Modern LLM
      ↓
One Playable Character
```

No production-readiness is implied.

Expect:

* incomplete functionality
* unstable APIs
* breaking changes
* failed experiments
* architectural revisions
* missing documentation

---

# Author

**KuangWei Chen**

---

# Closing Principle

> **ALMA came first.
> We think its ideas are worth carrying forward.**
>
> **The LLM gives the character a voice.
> Structured state may give the character continuity.**
>
> **AffectBridge exists to find out.**
