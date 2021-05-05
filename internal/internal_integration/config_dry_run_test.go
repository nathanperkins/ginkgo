package internal_integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/internal/test_helpers"
	. "github.com/onsi/gomega"
)

var _ = Describe("when config.DryRun is enabled", func() {
	BeforeEach(func() {
		conf.DryRun = true
		conf.SkipStrings = []string{"E"}

		RunFixture("dry run", func() {
			BeforeSuite(rt.T("before-suite"))
			BeforeEach(rt.T("bef"))
			Describe("container", func() {
				It("A", rt.T("A"))
				It("B", rt.T("B", func() { F() }))
				PIt("C", rt.T("C", func() { F() }))
				It("D", rt.T("D"))
				It("E", rt.T("D"))
			})
			AfterEach(rt.T("aft"))
			AfterSuite(rt.T("after-suite"))
		})
	})

	It("does not run any tests", func() {
		Ω(rt).Should(HaveTrackedNothing())
	})

	It("reports on the tests (both that they will run and that they did run) and honors skip state", func() {
		Ω(reporter.Will.Names()).Should(Equal([]string{"A", "B", "C", "D", "E"}))
		Ω(reporter.Will.Find("C")).Should(BePending())
		Ω(reporter.Will.Find("E")).Should(HaveBeenSkipped())

		Ω(reporter.Did.Names()).Should(Equal([]string{"A", "B", "C", "D", "E"}))
		Ω(reporter.Did.Find("A")).Should(HavePassed())
		Ω(reporter.Did.Find("B")).Should(HavePassed())
		Ω(reporter.Did.Find("C")).Should(BePending())
		Ω(reporter.Did.Find("D")).Should(HavePassed())
		Ω(reporter.Did.Find("E")).Should(HaveBeenSkipped())
	})

	It("reports the correct statistics", func() {
		Ω(reporter.End).Should(BeASuiteSummary(NSpecs(5), NPassed(3), NPending(1), NSkipped(1)))
	})
})