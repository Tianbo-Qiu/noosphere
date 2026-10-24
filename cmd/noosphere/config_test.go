package main

import (
	"time"

	"github.com/Tianbo-Qiu/noosphere/internal/web"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("loadConfig", func() {
	BeforeEach(func() {
		for _, key := range []string{"DATABASE_URL", "ADDR", "COOKIE_SECURE", "LOG_FORMAT", "LOG_LEVEL"} {
			GinkgoT().Setenv(key, "")
		}
	})

	It("uses the defaults when only DATABASE_URL is set", func() {
		GinkgoT().Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")

		cfg, err := loadConfig()

		Expect(err).NotTo(HaveOccurred())
		Expect(cfg).To(Equal(config{
			DatabaseURL: "postgres://test:test@localhost:5432/test",
			Addr:        ":8080",
			LogFormat:   "text",
			LogLevel:    "info",
			Web:         web.Config{SecureCookies: true, FlowLifetime: time.Hour},
		}))
	})

	It("reads every setting from its environment variable", func() {
		GinkgoT().Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
		GinkgoT().Setenv("ADDR", "127.0.0.1:9000")
		GinkgoT().Setenv("COOKIE_SECURE", "false")
		GinkgoT().Setenv("LOG_FORMAT", "json")
		GinkgoT().Setenv("LOG_LEVEL", "debug")

		cfg, err := loadConfig()

		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Addr).To(Equal("127.0.0.1:9000"))
		Expect(cfg.Web.SecureCookies).To(BeFalse())
		Expect(cfg.LogFormat).To(Equal("json"))
		Expect(cfg.LogLevel).To(Equal("debug"))
	})

	It("requires DATABASE_URL", func() {
		_, err := loadConfig()

		Expect(err).To(MatchError("DATABASE_URL is not set"))
	})

	DescribeTable("accepts the usual spellings of a boolean for COOKIE_SECURE",
		func(value string, want bool) {
			GinkgoT().Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
			GinkgoT().Setenv("COOKIE_SECURE", value)

			cfg, err := loadConfig()

			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Web.SecureCookies).To(Equal(want))
		},
		Entry("true", "true", true),
		Entry("1", "1", true),
		Entry("false", "false", false),
		Entry("0", "0", false),
		Entry("FALSE", "FALSE", false),
	)

	DescribeTable("rejects a COOKIE_SECURE that is not a boolean",
		func(value string) {
			GinkgoT().Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
			GinkgoT().Setenv("COOKIE_SECURE", value)

			_, err := loadConfig()

			Expect(err).To(MatchError(HavePrefix("COOKIE_SECURE: ")))
		},
		Entry("a word", "yes"),
		Entry("a typo", "ture"),
	)
})
