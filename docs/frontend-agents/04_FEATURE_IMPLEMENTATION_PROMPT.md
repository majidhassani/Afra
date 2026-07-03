# AgentVerse React Frontend - Feature Implementation Prompt

## Role

You are the Feature Implementation Agent for AgentVerse.

Implement all gameplay and business features end to end. Do not leave placeholder screens.

## Implementation Rule

Every feature must include:

- route
- data loader/query
- mutations
- loading state
- empty state
- error state
- success state
- responsive layout
- English translations
- Persian translations
- interaction polish

## Auth Feature

Screens:

- Login
- Register

Implement:

- form validation
- submit loading
- backend error handling
- token persistence
- redirect after login
- logout

## Profile Feature

Screens:

- Profile overview
- Stats
- Mission history
- Badges

Actions:

- edit display name

Show:

- rank
- level
- XP
- success rate
- total missions
- completed/failed missions
- favorite mission type
- total clues found
- total AI interactions
- total locations visited
- total coins spent/earned

## Wallet Feature

Screens:

- Wallet overview
- Pricing
- Transactions

Actions:

- rewarded ad claim
- purchase verify testing form

Show:

- balance
- reserved balance
- pricing rules
- cost warnings
- transaction list

## Missions Feature

Screens:

- Mission list
- Mission creation
- Mission dashboard
- Mission generation progress

Mission creation fields:

- type
- difficulty
- region
- language

Mission dashboard sections:

- briefing
- objectives
- public state
- current time
- quick map preview
- characters preview
- clues preview
- guidance composer
- event timeline

## Map Feature

Screens:

- Mission map
- Location detail panel

Actions:

- inspect area
- scan environment
- review documents
- ask AI
- view clues
- talk to character

Map states:

- discovered
- visited
- locked
- hidden locations must not render

Location detail must show:

- name
- type
- description
- risk level
- available actions
- characters
- discovered clues
- visual prompt

## Characters Feature

Screens:

- Character list
- Character detail
- Character chat

Show:

- name
- role
- category
- public profile
- mood
- trust level
- stress level
- dialogue style
- avatar prompt placeholder
- thumbnail prompt placeholder
- message history

Chat action:

```http
POST /api/v1/missions/{missionID}/characters/{characterID}/chat
```

Display result:

- reply message
- emotion
- mood
- trust delta
- stress delta
- unlocked clues
- new facts
- coins charged

## Clues Feature

Screens:

- Clue list
- Clue detail

Show:

- title
- type
- short description
- detailed description
- visual description
- prompt placeholder
- reliability
- importance
- public data

Actions:

- inspect
- explain

Inspect result:

- analysis
- new facts
- updated clue
- coins charged

Explain result:

- explanation
- next steps
- compare with
- coins charged

## Guidance Feature

Implement a reusable guidance composer:

- message textarea
- context selector
- submit
- response panel
- referenced items
- hint level
- coins charged

Use it in:

- mission dashboard
- map
- location detail
- clue detail
- journal
- time screen

## Time Feature

Screens:

- Mission time

Show:

- current time
- public state
- risk indicators

Actions:

- advance by minutes/hours/days

Result:

- new time
- summary
- events
- coins charged

## Journal Feature

Screens:

- Journal notes list
- Note editor

Actions:

- create
- update
- delete

Requirements:

- Persian and English typing support
- autosave indicator if implemented
- clear manual save if not autosave

## Events Feature

Show:

- mission events
- live stream state
- generation progress events
- event type icons
- event payload summary

Use SSE with fallback polling.

## Diagnostics Feature

Show:

- health
- ready
- API base URL
- auth session status
- backend latency
- last error

## Completion Checklist

The feature implementation is not done until all items above exist and work in both languages.
