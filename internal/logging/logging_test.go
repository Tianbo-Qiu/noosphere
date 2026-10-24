package logging

import (
	"bytes"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("New", func() {
	It("writes text lines in the text format", func() {
		var out bytes.Buffer
		logger, err := New(&out, "text", "info")
		Expect(err).NotTo(HaveOccurred())

		logger.Info("flow created", "flow_id", "01a12be8-3280-760b-a59b-6095af80070a")

		Expect(out.String()).To(ContainSubstring(`level=INFO msg="flow created" flow_id=01a12be8-3280-760b-a59b-6095af80070a`))
	})

	It("writes one JSON object per line in the json format", func() {
		var out bytes.Buffer
		logger, err := New(&out, "json", "info")
		Expect(err).NotTo(HaveOccurred())

		logger.Info("flow created", "flow_id", "01a12be8-3280-760b-a59b-6095af80070a")

		var line map[string]any
		Expect(json.Unmarshal(out.Bytes(), &line)).To(Succeed())
		Expect(line).To(HaveKeyWithValue("msg", "flow created"))
		Expect(line).To(HaveKeyWithValue("flow_id", "01a12be8-3280-760b-a59b-6095af80070a"))
	})

	It("drops lines below the level", func() {
		var out bytes.Buffer
		logger, err := New(&out, "text", "warn")
		Expect(err).NotTo(HaveOccurred())

		logger.Info("not shown")
		logger.Warn("shown")

		Expect(out.String()).NotTo(ContainSubstring("not shown"))
		Expect(out.String()).To(ContainSubstring(`msg=shown`))
	})

	DescribeTable("accepts each level",
		func(level string) {
			_, err := New(&bytes.Buffer{}, "text", level)

			Expect(err).NotTo(HaveOccurred())
		},
		Entry("debug", "debug"),
		Entry("info", "info"),
		Entry("warn", "warn"),
		Entry("error", "error"),
		Entry("uppercase", "INFO"),
	)

	It("rejects an unknown level", func() {
		_, err := New(&bytes.Buffer{}, "text", "loud")

		Expect(err).To(MatchError(HavePrefix("log level: ")))
	})

	It("rejects an unknown format", func() {
		_, err := New(&bytes.Buffer{}, "xml", "info")

		Expect(err).To(MatchError(`log format: unknown format "xml"`))
	})
})
