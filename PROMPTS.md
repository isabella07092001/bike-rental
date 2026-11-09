# Prompts

The main prompts used to build this project with Claude Code.

## Constitution

> We are creating a bike rental management system. Look in the README.md for input from stakeholders. Look in the TECH.md for the tech stack. Let's create a constitution in a specs directory: mission.md, tech-stack.md, roadmap-md.

## Phase 0 – Project Skeleton

> Start Phase 0.

## Phase 1 – Bike Management

> Create a new directory for Phase 1: bike-management.
In it: plan.md (series of numbered task groups) and requirements.md
(for the scope, decisions, context). Then ask me about this feature spec.

Answers to the questions:

| Question | Answer |
|---|---|
| Hourly prices for the four categories? | Placeholders (4 / 6 / 9 / 10 €) |
| Fields besides ID, name and category? | Status, Description / notes |
| What happens when a bike is deleted? | Retire / archive |
| Should bike names be unique? | Yes, unique |

> Start Phase 1

## Phase 2 – Person Management

> Create a new directory for Phase 2: person-management. In it: plan.md 
(series of numbered task groups) and requirements.md (for the scope, decisions, context).
Then ask me about this feature spec.

Answers to the questions:

| Question | Answer |
|---|---|
| Fields besides first name, last name and ID number? | Phone |
| When must the ID copy be uploaded? | At creation |
| Which files are allowed for the ID copy? | PDF/JPG/PNG ≤ 5 MB |
| What happens when a person is removed? | Archive like bikes |

> Start Phase 2

## Phase 3 – Reservation Management

> Create a new directory for Phase 3: reservation-management. In it: plan.md
(series of numbered task groups) and requirements.md (for the scope, decisions, context).
Then ask me about this feature spec.

Answers to the questions:

| Question | Answer |
|---|---|
| How should partial hours be billed? | Per started hour |
| Can one reservation contain several bikes? | Yes, several |
| Does a bike need a gap between two reservations? | 30 minutes |
| Can reservations be changed after booking? | Edit until 24 h before |

> Start Phase 3

## Phase 4 – Dashboard

> Create a new directory for Phase 4: dashboard. In it: plan.md
(series of numbered task groups) and requirements.md (for the scope, decisions, context).
Then ask me about this feature spec.

Answers to the questions:

| Question | Answer |
|---|---|
| Which sections should the dashboard show? | Fleet status now |
| How should revenue be counted? | No revenue |
| How far ahead should upcoming reservations be listed? | Next 7 days |
| Should the dashboard refresh by itself? | Only when opened |

> Start Phase 4

## Phase 5 – Polish

> Für Phase 5: pinkes/rosa Design, der Bike Shop heisst Power Bike,
Logo pinkes Fahrrad-Icon mit weiblichen Person.

> Go on with phase 5