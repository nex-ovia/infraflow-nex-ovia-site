+++
title   = "How I lead software delivery with AI agents and still own every line"
date    = 2026-11-24
summary = "AI can write the code. Leadership is deciding what gets built, proving it works, and keeping it understandable. Here is the operating model I use, and the overnight rebuild of this site that ran on it."
tags    = ["ai", "leadership", "nxs"]
draft   = true

[social]
linkedin  = "draft"
x         = "draft"
instagram = "off"
+++

AI can write the code now. That part is settled. What isn't settled is who owns the result: who understands it, who can change it safely next quarter, and who can move it when the vendor or the model changes.

That's the problem I wrote the [Nexovia Standard](/standard) to solve, and it's how I run my own delivery work every day.

## The operating model

I treat AI agents as a team I direct, not as autocomplete. Every piece of work goes through five steps:

1. **Plan.** I write the intent, constraints and acceptance checks in plain TOML before any code exists. The plan is the contract, and anyone can read it.
2. **Chunk.** The plan is cut into the smallest independent units: one function, one page, one config change. Each chunk carries its own validation gate.
3. **Dispatch.** A dispatcher sends each chunk to an AI worker in an isolated git worktree. Cheap, fast models take routine chunks; stronger ones take the hard ones.
4. **Validate.** Nothing is accepted on the model's word. Tests, builds and link checks run automatically, and a failure is isolated to one small chunk.
5. **Approve.** I review the diff and merge it. Production changes wait behind a human approval gate. The AI proposes; a person decides.

## A real example: this website

The version of this site you're reading was rebuilt in one night with that model. I wrote the plan and the copy; small AI workers did the edits, each in its own worktree, and each chunk had to pass the same gate before I merged it: Go tests, a full site build, and a check for broken links. Pages were then rendered at phone and desktop widths to catch layout problems.

Two things the gates caught that a confident model wouldn't have flagged:

- One worker "passed" its check by rewording a planning note instead of fixing the code. I rejected that part of the diff.
- The phone layout had been hiding the navigation entirely. The render check made it obvious, and the fix became one more small chunk.

Nothing went live until I approved it.

## Why it works for a leader

- **I'm not the bottleneck.** The agents do the typing; I do the deciding.
- **Speed doesn't turn into debt.** Every change is small, tested and reversible.
- **It's readable by non-developers.** The plan says what was built and why, in plain language. That's NXS rule four: it runs anywhere, for anyone.

The full model, with the guardrails, is on [How I lead with AI](/ai). The tools I build this way are on [Tools](/tools). If you want to talk about it on a podcast, in an interview or with your team, [get in touch](/work-with-me).
