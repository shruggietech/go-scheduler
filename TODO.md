# TODO

High-level roadmap. The authoritative, dependency-ordered task list lives in
[`specs/001-task-scheduler/tasks.md`](specs/001-task-scheduler/tasks.md) (78 tasks, 8 phases).

## Now — MVP (User Story 1)

- [ ] **Phase 1 · Setup** — Go module, dependencies, lint/CI, platform stubs (T001–T005)
- [ ] **Phase 2 · Foundational** — Clock, config, logging, SQLite store + base models, IPC,
      API skeleton, daemon (T006–T019)
- [ ] **Phase 3 · US1 (MVP)** — human-readable + one-off scheduling, DST, windowless executor,
      task CLI, system service / boot, runs & alerts query, `run_as`, cron-parity suite
      (T020–T037, T074–T078)
- [ ] **Validate MVP** — recurring + one-off tasks run windowless, persist, resume after restart

## Next — incremental delivery

- [ ] **Phase 4 · US2** — Material Design GUI: calendar/timeline, guided editor, live alerts (P2)
- [ ] **Phase 5 · US3** — nested task groups with cascading enable/disable (P2)
- [ ] **Phase 6 · US4** — event triggers (task completion) with at-least-once + dedup (P3)
- [ ] **Phase 7 · US5** — downtime catch-up (one run per task, then resume) (P3)

## Polish & cross-cutting

- [ ] README/build docs, goroutine-leak test, dispatch-latency benchmark (p99 < 100ms)
- [ ] Harden local IPC access control (socket/pipe permissions, admin group)
- [ ] Verify ≥80% coverage on core packages
- [ ] Cross-platform build incl. Windows windowless GUI + no-console task spawn
- [ ] Run quickstart end-to-end against all Success Criteria (SC-001…SC-010)

## Later / ideas (out of scope for v1)

- [ ] External trigger sources (CLI/API-delivered events) and file/folder watching
- [ ] Remote/multi-user GUI access
- [ ] External notification channels (email, push, webhooks)
- [ ] Distributed / multi-machine scheduling
