# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project context
This is a web app that allows users to select mountain resorts which they would like to be notified for when it snows. There are three main components, the frontend web app, the backend api layer, and a cronjob that will periodically check the weather in all resorts and fire alerts to users when a resort is prediected to have more than or equal to the amount of snow that the user signed up for. 

## Build/Test Commands
- Build: `make build`
- Run dev environment: `make dev`
- Install dependencies: `make install`
- Server tests: `cd server && go test ./...`

## Code Style Guidelines
- **Frontend (TypeScript/React)**:
  - Use tabs for indentation, no semicolons
  - Single quotes for strings
  - Explicit type annotations for function parameters/returns
  - Use React hooks for state management
  - Material UI for components
  - Use bun as a package manager

- **Backend (Go)**:
  - Standard Go code formatting (gofmt)
  - Explicit error handling with proper returns
  - Structured logging with context
  - Use http.Handler interface pattern
  - Security headers in all responses
  - Use sqlc for all sql items
  - Use goose for all db migrations

# Guide for coding
  - Please think about the solution before you begin coding, make sure you understand the request. If you need more information, please ask.
  - Get as much context about the problem by searching through the code base
  - Always run tests after changes to ensure no regressions are caused by your changes. If you write any new code, write new unit tests. 
  - When writing unit tests, make sure they are easy to understand and maintain. Only write the minimum amount of test to achieve a nice level of coverage.