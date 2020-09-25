# Changelog

## [v0.3.4](https://github.com/moorara/observer/tree/v0.3.4) (2020-09-25)

[Full Changelog](https://github.com/moorara/observer/compare/v0.3.3...v0.3.4)

**Merged pull requests:**

- Update http client example [\#70](https://github.com/moorara/observer/pull/70) ([moorara](https://github.com/moorara))
- Update OpenTelemetry modules to v0.12.0 [\#69](https://github.com/moorara/observer/pull/69) ([moorara](https://github.com/moorara))

## [v0.3.3](https://github.com/moorara/observer/tree/v0.3.3) (2020-09-23)

[Full Changelog](https://github.com/moorara/observer/compare/v0.3.2...v0.3.3)

**Closed issues:**

- Add RequestID to response header/metadata [\#62](https://github.com/moorara/observer/issues/62)

**Merged pull requests:**

- Add request id and client name to response headers [\#63](https://github.com/moorara/observer/pull/63) ([moorara](https://github.com/moorara))

## [v0.3.2](https://github.com/moorara/observer/tree/v0.3.2) (2020-09-19)

[Full Changelog](https://github.com/moorara/observer/compare/v0.3.1...v0.3.2)

**Fixed bugs:**

- Create noop logger, meter, and tracer by default [\#60](https://github.com/moorara/observer/issues/60)

**Merged pull requests:**

- Fix the bug related to nil logger, meter, and tracer [\#61](https://github.com/moorara/observer/pull/61) ([moorara](https://github.com/moorara))

## [v0.3.1](https://github.com/moorara/observer/tree/v0.3.1) (2020-09-10)

[Full Changelog](https://github.com/moorara/observer/compare/v0.3.0...v0.3.1)

**Fixed bugs:**

- Fix the bug for reporting number of in-flight requests [\#58](https://github.com/moorara/observer/issues/58)

**Merged pull requests:**

- Recovering and reporting panics from handlers [\#59](https://github.com/moorara/observer/pull/59) ([moorara](https://github.com/moorara))
- Update module google.golang.org/grpc to v1.32.0 [\#57](https://github.com/moorara/observer/pull/57) ([renovate[bot]](https://github.com/apps/renovate))
- Update module go.uber.org/zap to v1.16.0 [\#56](https://github.com/moorara/observer/pull/56) ([renovate[bot]](https://github.com/apps/renovate))
- Update module google/uuid to v1.1.2 [\#55](https://github.com/moorara/observer/pull/55) ([renovate[bot]](https://github.com/apps/renovate))

## [v0.3.0](https://github.com/moorara/observer/tree/v0.3.0) (2020-08-30)

[Full Changelog](https://github.com/moorara/observer/compare/v0.2.3...v0.3.0)

**Closed issues:**

- Provide an option to use OTEL collector [\#52](https://github.com/moorara/observer/issues/52)
- Consider renaming options [\#33](https://github.com/moorara/observer/issues/33)

**Merged pull requests:**

- Otel collector [\#54](https://github.com/moorara/observer/pull/54) ([moorara](https://github.com/moorara))
- Upgrade otel [\#53](https://github.com/moorara/observer/pull/53) ([moorara](https://github.com/moorara))
- Update module google.golang.org/grpc to v1.31.1 [\#51](https://github.com/moorara/observer/pull/51) ([renovate[bot]](https://github.com/apps/renovate))
- Housekeeping [\#46](https://github.com/moorara/observer/pull/46) ([moorara](https://github.com/moorara))

## [v0.2.3](https://github.com/moorara/observer/tree/v0.2.3) (2020-07-31)

[Full Changelog](https://github.com/moorara/observer/compare/v0.2.2...v0.2.3)

**Merged pull requests:**

- Update repo [\#45](https://github.com/moorara/observer/pull/45) ([moorara](https://github.com/moorara))
- Update module google.golang.org/grpc to v1.31.0 [\#44](https://github.com/moorara/observer/pull/44) ([renovate[bot]](https://github.com/apps/renovate))
- Update module go.opentelemetry.io/otel to v0.9.0 [\#42](https://github.com/moorara/observer/pull/42) ([renovate[bot]](https://github.com/apps/renovate))
- Update module go.opentelemetry.io/otel to v0.8.0 [\#41](https://github.com/moorara/observer/pull/41) ([renovate[bot]](https://github.com/apps/renovate))

## [v0.2.2](https://github.com/moorara/observer/tree/v0.2.2) (2020-07-03)

[Full Changelog](https://github.com/moorara/observer/compare/v0.2.1...v0.2.2)

**Merged pull requests:**

- Update OpenTelemetry module to v0.7.0 [\#40](https://github.com/moorara/observer/pull/40) ([moorara](https://github.com/moorara))
- Update module google.golang.org/protobuf to v1.25.0 [\#38](https://github.com/moorara/observer/pull/38) ([renovate[bot]](https://github.com/apps/renovate))
- Update module prometheus/client\_golang to v1.7.1 [\#37](https://github.com/moorara/observer/pull/37) ([renovate[bot]](https://github.com/apps/renovate))
- Update module google.golang.org/grpc to v1.30.0 [\#36](https://github.com/moorara/observer/pull/36) ([renovate[bot]](https://github.com/apps/renovate))
- Add Go Doc README badges [\#35](https://github.com/moorara/observer/pull/35) ([moorara](https://github.com/moorara))

## [v0.2.1](https://github.com/moorara/observer/tree/v0.2.1) (2020-06-18)

[Full Changelog](https://github.com/moorara/observer/compare/v0.2.0...v0.2.1)

**Merged pull requests:**

- Update README.md [\#32](https://github.com/moorara/observer/pull/32) ([moorara](https://github.com/moorara))

## [v0.2.0](https://github.com/moorara/observer/tree/v0.2.0) (2020-06-18)

[Full Changelog](https://github.com/moorara/observer/compare/v0.1.0...v0.2.0)

**Implemented enhancements:**

- ohttp: report canonical url paths when reporting metrics [\#17](https://github.com/moorara/observer/issues/17)

**Fixed bugs:**

- Prometheus histogram metrics cannot be created [\#9](https://github.com/moorara/observer/issues/9)
- Prometheus metrics do not have labels [\#8](https://github.com/moorara/observer/issues/8)

**Closed issues:**

- Finish ogrpc package [\#27](https://github.com/moorara/observer/issues/27)
- Finish ohttp package [\#20](https://github.com/moorara/observer/issues/20)
- Add a request gauge instrument [\#16](https://github.com/moorara/observer/issues/16)

**Merged pull requests:**

- Add examples [\#31](https://github.com/moorara/observer/pull/31) ([moorara](https://github.com/moorara))
- add traceId and spanId to context logger [\#30](https://github.com/moorara/observer/pull/30) ([moorara](https://github.com/moorara))
- Add unit tests [\#29](https://github.com/moorara/observer/pull/29) ([moorara](https://github.com/moorara))
- Add grpc [\#26](https://github.com/moorara/observer/pull/26) ([moorara](https://github.com/moorara))
- Update module stretchr/testify to v1.6.1 [\#25](https://github.com/moorara/observer/pull/25) ([renovate[bot]](https://github.com/apps/renovate))
- Update module stretchr/testify to v1.6.0 [\#24](https://github.com/moorara/observer/pull/24) ([renovate[bot]](https://github.com/apps/renovate))
- Update feature\_request.md [\#22](https://github.com/moorara/observer/pull/22) ([moorara](https://github.com/moorara))
- Add ohttp sub-package [\#19](https://github.com/moorara/observer/pull/19) ([moorara](https://github.com/moorara))
- Change Context API [\#15](https://github.com/moorara/observer/pull/15) ([moorara](https://github.com/moorara))
- Remove summary quantiles and histogram options from options [\#14](https://github.com/moorara/observer/pull/14) ([moorara](https://github.com/moorara))
- Update module prometheus/client\_golang to v1.6.0 [\#13](https://github.com/moorara/observer/pull/13) ([renovate[bot]](https://github.com/apps/renovate))

## [v0.1.0](https://github.com/moorara/observer/tree/v0.1.0) (2020-04-24)

[Full Changelog](https://github.com/moorara/observer/compare/b854e571647301ebf995530765781fe0ea555904...v0.1.0)

**Merged pull requests:**

- Refactor API and update README [\#6](https://github.com/moorara/observer/pull/6) ([moorara](https://github.com/moorara))
- Add Observer API [\#4](https://github.com/moorara/observer/pull/4) ([moorara](https://github.com/moorara))
- Configure Renovate [\#3](https://github.com/moorara/observer/pull/3) ([renovate[bot]](https://github.com/apps/renovate))
- Add GitHub actions, templates, and misc files [\#2](https://github.com/moorara/observer/pull/2) ([moorara](https://github.com/moorara))
- Bootstrap 🚀 [\#1](https://github.com/moorara/observer/pull/1) ([moorara](https://github.com/moorara))



\* *This Changelog was automatically generated by [github_changelog_generator](https://github.com/github-changelog-generator/github-changelog-generator)*
