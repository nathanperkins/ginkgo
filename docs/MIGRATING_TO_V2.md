---
layout: default
title: Migrating to Ginkgo V2
---

[Ginkgo 2.0](#711) is a major release that adds substantial new functionality and removes/moves existing functionality.  **Please provide feedback, concerns, questions, etc. on issue [#711](#711).**

This document serves as a changelog and migration guide for users migrating from Ginkgo 1.x to 2.0.  The intent is that the migration will take minimal user effort.

The work on 2.0 is tracked in this [Pivotal Tracker backlog](https://www.pivotaltracker.com/n/projects/2493102).  The original 2.0 proposal is [here](https://docs.google.com/document/d/1h28ZknXRsTLPNNiOjdHIO-F2toCzq4xoZDXbfYaBdoQ/edit#heading=h.mzgqmkg24xoo) however this is starting to grow stale.  The tracker backlog is the most authoritative version of the roadmap.

The current timeline for completion of 2.0 looks like:

- Early April 2021: first public release of 2.0, deprecation warnings land in v1.
- May 2021: first beta/rc of 2.0 with most new functionality in place.
- June/July 2021: 2.0 ships and fully replaces the 1.x codebase on master.

But is, of course, subject to change ;)

# Additions and Improvements

## Major Additions and Improvements

### Interrupt Behavior
Interrupt behavior is substantially improved, sending an interrupt signal will now:
	- immediately cause the current test to unwind.  Ginkgo will run any `AfterEach` blocks, then immediately skip all remaining tests, then run the `AfterSuite` block.  
	- emit information about which node Ginkgo was running when the interrupt signal was recevied.
	- emit as much information as possible about the interrupted test (e.g. `GinkgoWriter` contents, `stdout` and `stderr` context).
	- emit a stack trace of every running goroutine at the moment of interruption.

Previously, sending a second interrupt signal would cause Ginkgo to exit immediately.  With the improved interrupt behavior this is no longer necessary and Ginkgo will not exit until the test suite has unwound and completed.

### Timeout Behavior
In Ginkgo V1.x, Gingko's timeout was managed by `go test`.  This meant that timeouts exited the test suite abruptly with no opportunity for custom reporters or clean up code (e.g. `AfterEach`, `AfterSuite`) to run.  This is fixed in V2.  Ginkgo now manages its own timeout and when a timeout triggers the test winds down gracefully.  In fact, a timeout is now functionally equivalen to a user-initiated interrupt.

In addition, in V1.x when running multiple test suites Ginkgo would give each suite the full timeout allotment (so `ginkgo -r -timeout=1h` would give _each_ test suite one hour to complete).  In V2 the timeout now applies to the entire test suite run so that `ginkgo -r -timeout=1h` is now guaranteed to exit after (about) one hour.

**Finally, the default timeout has been reduced from `24h` down to `1h`.**  Users with long-running tests may need to adjust the timeout in their CI scripts.

### Spec Decorations
Specs can now be decorated with a series of new spec decorators.  These decorators enable fine-grained control over certain aspects of the spec's creation and lifecycle. 

To support decorations, the signature for Ginkgo's container, setup, and It nodes have been changed to:
```
func Describe(text string, args ...interface{})
func It(text string, args ...interface{})
func BeforeEach(args ...interface{})
```
Note that this change is backwards compatible with v1.X.

Ginkgo supports passing in decorators _and_ arbitrarily nested slices of decorators.  Ginkgo will unroll any slices and process the flattened list of decorators.  This makes it easier to pass around and combine groups of decorators.

Here's a list of new decorators.  They are documented in more detail in the [Node Decoration Reference](https://github.com/onsi/ginkgo/blob/v2/docs/index.md#node-decoration-reference) section of the documentation.

#### Focus Decoration
In addition to `FDescribe` and `FIt`, specs can now be focused using the new `Focus` decoration:

```
Describe("a focused container", Focus, func() {
  ....
})
```

#### Pending Decoration
In addition to `PDescribe` and `PIt`, specs can now be focused using the new `Focus` decoration:

```
Describe("a focused container", Pending, func() {
  ....
})
```

#### Offset Decoration
The `Offset(uint)` decoration allows the user to change the stack-frame offset used to compute the location of the test node.  This is useful when building shared test behaviors.  For example:

```
SharedBehaviorIt := func() {
	It("does something common and complicated", Offset(1), func() {
		...
	})
}

Describe("thing A", func() {
	SharedBehaviorIt()
})

Describe("thing B", func() {
	SharedBehaviorIt()
})
```

now, if the `It` defined in `SharedBehaviorIt` the location reported by Ginkgo will point to the line where `SharedBehaviorIt` is *invoked*.

#### FlakeAttempts Decoration
The `FlakeAttempts(uint)` decoration allows the user to flag specific tests or groups of tests as potentially flaky.  Ginkgo will run tests up to the number of times specified in `FlakeAttempts` until they pass.  For example:

```
Describe("flaky tests", FlakeAttempts(3), func() {
	It("is flaky", func() {
		...
	})

	It("is also flaky", func() {
		...
	})

	It("is _really_ flaky", FlakeAttempts(5) func() {
		...
	})

	It("is _not_ flaky", FlakeAttempts(1), func() {
		...
	})
})
```

With this setup, `"is flaky"` and `"is also flaky"` will run up to 3 times.  `"is _really_ flaky"` will run up to 5 times.  `"is _not_ flaky"` will run only once.

Note that if `ginkgo --flake-attempts=N` is set the value passed in by the CLI will override all the decorated values.  Every test will now run up to `N` times.


### CLI Flags
Ginkgo's CLI flags have been rewritten to provide clearer, better-organized documentation.  In addition, Ginkgo v1 was mishandling several go cli flags.  This is now resolved with clear distinctions between flags intended for compilation time and run-time.  As a result, users can now generate `memprofile`s and `cpuprofile`s using the Ginkgo CLI.  Ginkgo 2.0 will automatically merge profiles generated by running tests in parallel (i.e. across multiple processes) and will allow you to choose between having profiles stored in individual package directories, or collected in one place using the `-output-dir` flag.  See [Changed: Profiling Support](#changed-profiling-support) for more details.

### Expanded GinkgoWriter Functionality
The `GinkgoWriter` is used to write output that is only made visible if a test fails, or if the user runs in verbose mode with `ginkgo -v`.

In Ginkgo 2.0 `GinkgoWriter` now has:
	- Three new convenience methods `GinkgoWriter.Print(a ...interface{})`, `GinkgoWriter.Println(a ...interface{})`, `GinkgoWriter.Printf(format string, a ...interface{})`  These are equivalent to calling the associated `fmt.Fprint*` functions and passing in `GinkgoWriter`.
	- The ability to tee to additional writers.  `GinkgoWriter.TeeTo(writer)` will send any future data written to `GinkgoWriter` to the passed in `writer`.  You can attach multiple `io.Writer`s for `GinkgoWriter` to tee to.  You can remove all attached writers with `GinkgoWriter.ClearTeeWriters()`.

	Note that _all_ data written to `GinkgoWriter` is immediately forwarded to attached tee writers regardless of where a test passes or fails.

### Improved: Reporting Infrastructure
Ginkgo V2 provides an improved reporting infrastructure that [replaces and improves upon Ginkgo V1's support for custom reporters](#removed-custom-reporters).  Here are a few distinct use-cases that the new reporting infrastructure supports:

#### Generating machine-readable reports
Ginkgo now natively supports generating and aggregating reports in a number of machine-readable formats - and these reports can be generated and managed by simply passing `ginkgo` command line flags.

Ginkgo V2 introduces a new JSON format that faithfully captures all avialable information about a Ginkgo test suite.  JSON reports can be generated via `ginkgo --json-report=out.json`.  The resulting JSON file encodes an array of `types.Report`.  Each entry in that array lists detailed information about the test suite and includes a list of `types.SpecReport` that captures detailed information about each spec.  These types are documented [here](https://github.com/onsi/ginkgo/blob/v2/types/types.go).

Ginkgo also supports generating JUnit reports with `ginkgo --junit-report=out.xml` and Teamcity reports with `ginkgo --teamcity-report=out.teamcity`.  In addition, Ginkgo V2's JUnit reporter has been improved and is now more conformant with the JUnit specification.

Ginkgo follows the following rules when generating reports using these new `--FORMAT-report` flags:
- By default, a single report file per format is generated at the passed-in file name.  This single report merges all the reports generated by each individual suite.
- If `-output-dir` is set: the report files are placed in the specified `-output-dir` directory.
- If `-keep-separate-reports` is set, the individual reports generated by each test suite are not merged.  Instead, individual report files will appear in each package directory.  If `-output-dir` is _also_ set these individual files are copied into the `-output-dir` directory and namespaced with `PACKAGE_NAME_{REPORT}`.

#### Generating Custom Reports when a test suite completes
Ginkgo now provides a new node, `ReportAfterSuite`, with the following properties and constraints:
- `ReportAfterSuite` nodes are passed a function that takes a `types.Report`:
  ```
  var _ = ReportAfterSuite(func(report types.Report) {
  	// do stuff with report
  })
  ```
- These functions are called exactly once at the end of the test suite after any `AfterSuite` nodes have run.  When running in parallel the functions are called on the primary Ginkgo process after all other processes have finished and is guaranteed to have an aggregated copy of `types.Report` that includes all `SpecReport`s from all the Ginkgo parallel processes.
- If a failure occurs in `ReportAfterSuite` it is reported in reports sent to subsequent `ReportAfterSuite`s.  In particular, it is reported as part of Ginkgo's default output and is in included in any reports generated by the `--FORMAT-report` flags described above.
- `ReportAfterSuite` nodes **cannot** be interrupted by the user.  This is to ensure the integrity of generated reports... so be careful what kind of code you put in there!
- Multiple `ReportAfterSuite` nodes can be registered per test suite, but all must be defined at the top-level of the suite.

`ReportAfterSuite` is useful for users who want to emit a custom-formatted report or register the results of the test run with an external service.

#### Capturing report information about each spec as the test suite runs
Ginkgo also provides a new node, `ReportAfterEach`, with the following properties and constraints:
- `ReportAfterEach` nodes are passed a function that takes a `types.SpecReport`:
  ```
  var _ = ReportAfterEach(func(specReport types.SpecReport) {
  	// do stuff with specReport
  })
  ```
- `ReportAfterEach` nodes are called after a spec completes (i.e. after any `AfterEach` nodes have run).  `ReportAfterEach` nodes are **always** called - even if the test has failed, is marked pending, or is skipped.  In this way, the user is guaranteed to have access to a report about every spec defined in a suite.
- If a failure occurs in `ReportAfterEach`, the spec in question is marked as failed.  Any subsequently defined `ReportAfterEach` block will receive an updated report that includes the failure.  In general, though, assertions about your code should go in `AfterEach` nodes.
- `ReportAfterEach` nodes **cannot** be interrupted by the user.  This is to ensure the integrity of generated reports... so be careful what kind of code you put in there!
- `ReportAfterEach` nodes can be placed in any container node in the suite's hierarchy - in this way the follow the same semantics as `AfterEach` blocks.
- When running in parallel, `ReportAfterEach` nodes will run on the Ginkgo process that is running the spec being reported on.  This means that multiple `ReportAfterEach` blocks can be running concurrently on independent processes.

`ReportAfterEach` is useful if you need to stream or emit up-to-date information about the test suite as it runs.

### Improved: Profiling Support
Ginkgo V1 was incorrectly handling Go test's various profiling flags (e.g. -cpuprofile, -memprofile).  This has been fixed in V2.  In fact, V2 can capture profiles for multiple packages (e.g. ginkgo -r -cpuprofile=profile.out will work).

When generating profiles for `-cpuprofile=FILE`, `-blockprofile=FILE`, `-memprofile=FILE`, `-mutexprofile=FILE`, and `-execution-trace=FILE` (Ginkgo's alias for `go test -test.trace`) the following rules apply:

- If `-output-dir` is not set: each profile generates a file named `$FILE` in the directory of each package under test.
- If `-output-dir` is set: each profile generates a file in the specified `-output-dir` directory. named `PACKAGE_NAME_$FILE`

### Improved: Cover Support
Coverage reporting is much improved in 2.0:

- `ginkgo -cover -p` now emits code coverage after the test completes, just like `ginkgo -cover` does in series.
- When running across mulitple packages (e.g. `ginkgo -r -cover`) ginkgo will now emit a composite coverage statistic that represents the total coverage across all test suites run.  (Note that this is disabled if you set `-keep-separate-coverprofiles`).

In addition, Ginkgo now follows the following rules when generating cover profiles using `-cover` and/or `-coverprofile=FILE`:

- By default, a single cover profile is generated at `FILE` (or `coverprofile.out` if `-cover-profile` is not set but `-cover` is set).  This includes the merged results of all the cover profiles reported by each suite.
- If `-output-dir` is set: the `FILE` is placed in the specified `-output-dir` directory.
- If `-keep-separate-coverprofiles` is set, the individual coverprofiles generated by each package are not merged and, instead, a file named `FILE` will appear in each package directory.  If `-output-dir` is _also_ set these files are copied into the `-output-dir` directory and namespaced with `PACKAGE_NAME_{FILE}`.

### New: --repeat
Ginkgo can now repeat a test suite N additional times by running `ginkgo --repeat=N`.  This is similar to `go test -count=N+1` and is a variant of `ginkgo --until-it-fails` that can be run in CI environments to repeat test runs to suss out flakey tests.

Ginkgo requires the tests to succeed during each repetition in order to consider the test run a success.

## Minor Additions and Improvements
- `BeforeSuite` and `AfterSuite` no longer run if all tests in a suite are skipped.
- Ginkgo can now catch several common user gotchas and emit a helpful error.
- Error output is clearer and consistently links to relevant sections in the documentation.
- Test randomization is now more stable as tests are now sorted deterministcally on file_name:line_number first (previously they were sorted on test text which could not guarantee a stable sort).

# Changes

## Major Changes
These are major changes that will need user intervention to migrate succesfully.

### Removed: Async Testing
As described in the [Ginkgo 2.0 Proposal](https://docs.google.com/document/d/1h28ZknXRsTLPNNiOjdHIO-F2toCzq4xoZDXbfYaBdoQ/edit#heading=h.mzgqmkg24xoo) the Ginkgo 1.x implementation of asynchronous testing using a `Done` channel was a confusing source of test-pollution.  It is removed in Ginkgo 2.0.

In Ginkgo 2.0 tests of the form:

```
It("...", func(done Done) {
	// user test code to run asynchronously
	close(done) //signifies the test is done
}, timeout)
```

will emit a deprecation warning and will run **synchronously**.  This means the `timeout` will not be enforced and the status of the `Done` channel will be ignored - a test that hangs will hang indefinitely.

#### Migration Strategy:
We recommend users make targeted use of Gomega's [Asynchronous Assertions](https://onsi.github.io/gomega/#making-asynchronous-assertions) to better test asynchronous behavior.

As a first migration pass that produces **equivalent behavior** users can replace asynchronous tests with:

```
It("...", func() {
	done := make(chan interface{})
	go func() {
		// user test code to run asynchronously
		close(done) //signifies the code is done
	}()
	Eventually(done, timeout).Should(BeClosed())
})
```

### Removed: Measure
As described in the [Ginkgo 2.0 Proposal](https://docs.google.com/document/d/1h28ZknXRsTLPNNiOjdHIO-F2toCzq4xoZDXbfYaBdoQ/edit#heading=h.2ezhpn4gmcgs) the Ginkgo 1.x implementation of benchmarking using `Measure` nodes was a source of tightly-coupled complexity.  It is removed in Ginkgo 2.0.

In Ginkgo 2.0 tests of the form:
```
Measure(..., func(b Benchmarker) {
	// user benchmark code
})
```

will emit a deprecation warning and **will no longer run**.

#### Migration Strategy:
A new Gomega benchmarking subpackage is being developed to replace Ginkgo's benchmarking capabilities with a more mature, decoupled, and useful implementation.  This section will be updated once the Gomega package is ready.

### Removed: Custom Reporters
Ginkgo 2.0 removes support for Ginkgo 1.X's custom reporters - they behaved poorly when running in parallel and represented unnecessary and error-prone boiler plate for users who simply wanted to produce machine-readable reports.  Instead, the reporting infrastructure has been significantly improved to enable simpler support for the most common use-cases and custom reporting needs.

Please read through the [Improved: Reporting Infrastructure](#improved-reporting-infrastructure) section to learn more.  For users with custom reporters, follow the migration guide below.

#### Migration Strategy:
In Ginkgo 2.0 both `RunSpecsWithDefaultAndCustomReporters` and `RunSpecsWithCustomReporters` have been deprecated.  Users must call `RunSpecs` instead.

If you were using custom reporters to generate JUnit or Teamcity reports, simply call `RunSpecs` and [invoke your tests with the new `--junit-report` and/or `--teamcity-report` flags](#generating-machine-readable-reports).  Note that unlike the 1.X JUnit and Teamcity reporters, these flags generate unified reports for all test suites run (though you can adjust this with the `--keep-separate-reports` flag) and take care of aggregating reports from parallel processes for you.

If you've written your own custom reporter, [add a `ReportAfterSuite` node](#generating-custom-reports-when-a-test-suite-completes) and process the `types.Report` that it provides you.  If you'd like to continue using your custom reporter you can simply call `reporters.ReportViaDeprecatedReporter(reporter, report)` in `ReportAfterSuite` - though we recommend actually changing your code's logic to use the `types.Report` object directly as `reporters.ReportViaDeprecatedReporter` will be removed in a future release of Ginkgo 2.X.  Unlike 1.X custom reporters which are called concurrently by independent parallel processes when running in parallel, `ReportAFterSuite` is called exactly once per suite and is guaranteed to have aggregated information from all parallel processes.

Alternatively, you can use the new `--json-report` flag to produce a machine readable JSON-format report that you can post-process after the test completes.

Finally, if you still need the real-time reporting capabilities that 1.X's custom reporters provided you can use [`ReportAfterEach`](#capturing-report-information-about-each-spec-as-the-test-suite-runs) to get information about each spec as it completes.

### Changed: CurrentGinkgoTestDescription()
`CurrentGinkgoTestDescription()` has been deprecated and will be removed in a future release.  The method was returning a processed object that included a subset of information available about the running test.

It has been replaced with `CurrentSpecReport()` which returns the full-fledge `types.SpecReport` used by Ginkgo's reporting infrastructure.  To help users migrate, `types.SpecReport` now includes a number of helper methods to make it easier to extract information about the running test.

#### Migration Strategy:
Replace any calls to `CurrentGinkgoTestDescription()` with `CurrentSpecReport()` and use the struct fields or helper methods on the returned `types.SpecReport` to get the information you need about the current test.

### Changed: availability of Ginkgo's configuration
In v1 Ginkgo's configuration could be accessed by importing the `config` package and accessing the globally available `GinkgoConfig` and `DefaultReporterConfig` objects.  This is no longer supported in V2.

#### Migration Strategy:
Instead, configuration can be accessed using the DSL's `GinkgoConfiguration()` function.  This will return a `types.SuiteConfig` and `types.ReporterConfig`.  Users generally don't need to access this configuration - the most commonly used fields by end users are already made available via `GinkgoRandomSeed()` and `GinkgoParallelNode()`.

### Changed: Command Line Flags
All camel case flags (e.g. `-randomizeAllSpecs`) are replaced with kebab case flags (e.g. `-randomize-all-specs`) in Ginkgo 2.0.  The camel case versions continue to work but emit a deprecation warning.

#### Migration Strategy:
Users should update any scripts they have that invoke the `ginkgo` cli from camel case to kebab case (:camel: :arrow_right: :oden:).

### Removed: `-stream`
`-stream` was originally introduce in Ginkgo 1.x to force parallel test processes to emit output simultaneously in order to help debug hanging test issues.  With improvements to Ginkgo's interrupt handling and parallel test reporting this behavior is no longer necessary and has been removed.

### Removed: `-notify`
`-notify` instructed Ginkgo to emit desktop notifications on linux and MacOS.  This feature was rarely used and has been removed.

### Removed: `-noisyPendings` and `-noisySkippings`
Both these flags tweaked the reporter's behavior for pending and skipped tests.  A similar, and more intuitive, outcome is possible using `--succinct` and `-v`.

### Changed: `-slowSpecThreshold`
`-slowSpecThreshold` is now `-slow-spec-threshold` and takes a `time.Duration` (e.g. `5s` or `3m`) instead of a `float64` number of seconds.

### Removed: `-debug`
The `-debug` flag has been removed.  It functioned primarily as a band-aid to Ginkgo V1's poor handling of stuck parallel tests. The new [interrupt behavior](#interrupt-behavior) in V2 resolves the root issues behind the `-debug` flag.

#### Migration Strategy:
Users should remove -stream from any scripts they have that invoke the `ginkgo` cli.

### Removed: `ginkgo convert`
The `ginkgo convert` subcommand in V1 could convert an existing set of Go tests into a Ginkgo test suite, wrapping each `TestX` function in an `It`.  This subcommand added complexity to the codebase and was infrequently used.  It has been removed.  Users who want to convert tests suites over to Ginkgo will need to do so by hand.

## Minor Changes
These are minor changes that will be transparent for most users.

- `"top level"` is no longer the first element in `types.SpecReport.NodeTexts`.  This will only affect users who write custom reporters.

- The output format of Ginkgo's Default Reporter has changed in numerous subtle ways to improve readability and the user experience.  Users who were scraping Ginkgo output programatically may need to change their scripts or use the new JSON formatted report option (TODO: update with link once JSON reporting is implemented).

- When running in series and verbose mode (i.e. `ginkgo -v`) GinkgoWriter output is emitted in real-time (existing behavior) but also emitted in the failure message for failed tests.  This allows for consistent failure messages regardless of verbosity settings and also makes it possible for the resulting JSON report to include captured GinkgoWriter information.

- Removed `ginkgo blur` alias.  Use `ginkgo unfocus` instead.