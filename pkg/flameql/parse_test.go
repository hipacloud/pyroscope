package flameql

import (
	"errors"
	"regexp"
	"regexp/syntax"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseQuery", func() {
	It("parses queries", func() {
		type testCase struct {
			query string
			err   error
			q     *Query
		}

		testCases := []testCase{
			{`app.name`, nil, &Query{AppName: "app.name", q: `app.name`}},
			{`app.name{}`, nil, &Query{AppName: "app.name", q: `app.name{}`}},
			{`app.name{foo="bar"}`, nil,
				&Query{"app.name", []*TagMatcher{{"foo", "bar", EQL, nil}}, `app.name{foo="bar"}`}},
			{`app.name{foo="bar,baz"}`, nil,
				&Query{"app.name", []*TagMatcher{{"foo", "bar,baz", EQL, nil}}, `app.name{foo="bar,baz"}`}},
			{`app.name{foo="bar",baz!="quo"}`, nil,
				&Query{"app.name", []*TagMatcher{{"baz", "quo", NEQ, nil}, {"foo", "bar", EQL, nil}}, `app.name{foo="bar",baz!="quo"}`}},

			{"", ErrAppNameIsRequired, nil},
			{"{}", ErrAppNameIsRequired, nil},
			{`app.name{,}`, ErrInvalidMatchersSyntax, nil},
			{`app.name[foo="bar"]`, ErrInvalidAppName, nil},
			{`app=name{}"`, ErrInvalidAppName, nil},
			{`app.name{foo="bar"`, ErrInvalidQuerySyntax, nil},
			{`app.name{__name__="foo"}`, ErrTagKeyReserved, nil},
		}

		for _, tc := range testCases {
			q, err := ParseQuery(tc.query)
			if tc.err != nil {
				Expect(errors.Is(err, tc.err)).To(BeTrue())
			} else {
				Expect(err).To(BeNil())
			}
			Expect(q).To(Equal(tc.q))
		}
	})
})

var _ = Describe("ParseMatcher", func() {
	It("parses tag matchers", func() {
		type testCase struct {
			expr string
			err  error
			m    *TagMatcher
		}

		testCases := []testCase{
			{expr: `foo="bar"`, m: &TagMatcher{"foo", "bar", EQL, nil}},
			{expr: `foo="z"`, m: &TagMatcher{"foo", "z", EQL, nil}},
			{expr: `foo=""`, err: ErrInvalidTagValueSyntax},
			{expr: `foo="`, err: ErrInvalidTagValueSyntax},
			{expr: `foo="z`, err: ErrInvalidTagValueSyntax},
			{expr: `foo=`, err: ErrInvalidTagValueSyntax},

			{expr: `foo!="bar"`, m: &TagMatcher{"foo", "bar", NEQ, nil}},
			{expr: `foo!="z"`, m: &TagMatcher{"foo", "z", NEQ, nil}},
			{expr: `foo=~""`, err: ErrInvalidTagValueSyntax},
			{expr: `foo=~"`, err: ErrInvalidTagValueSyntax},
			{expr: `foo=~"z`, err: ErrInvalidTagValueSyntax},
			{expr: `foo=~`, err: ErrInvalidTagValueSyntax},

			{expr: `foo=~"bar"`, m: &TagMatcher{"foo", "bar", EQL_REGEX, nil}},
			{expr: `foo=~"z"`, m: &TagMatcher{"foo", "z", EQL_REGEX, nil}},
			{expr: `foo!=""`, err: ErrInvalidTagValueSyntax},
			{expr: `foo!="`, err: ErrInvalidTagValueSyntax},
			{expr: `foo!="z`, err: ErrInvalidTagValueSyntax},
			{expr: `foo!=`, err: ErrInvalidTagValueSyntax},

			{expr: `foo!~"bar"`, m: &TagMatcher{"foo", "bar", NEQ_REGEX, nil}},
			{expr: `foo!~"z"`, m: &TagMatcher{"foo", "z", NEQ_REGEX, nil}},
			{expr: `foo!~""`, err: ErrInvalidTagValueSyntax},
			{expr: `foo!~"`, err: ErrInvalidTagValueSyntax},
			{expr: `foo!~"z`, err: ErrInvalidTagValueSyntax},
			{expr: `foo!~`, err: ErrInvalidTagValueSyntax},

			{expr: `foo="bar,baz"`, m: &TagMatcher{"foo", "bar,baz", EQL, nil}},
			{expr: `foo="bar\",\"baz"`, m: &TagMatcher{"foo", "bar\\\",\\\"baz", EQL, nil}},

			{expr: `foo;bar="baz"`, err: ErrInvalidTagKey},
			{expr: `foo""`, err: ErrInvalidTagKey},
			{expr: `foo`, err: ErrMatchOperatorIsRequired},
			{expr: `foo!`, err: ErrInvalidTagValueSyntax},
			{expr: `foo!!`, err: ErrInvalidTagValueSyntax},
			{expr: `foo!~@b@"`, err: ErrInvalidTagValueSyntax},
			{expr: `foo=bar`, err: ErrInvalidTagValueSyntax},
			{expr: `foo!"bar"`, err: ErrUnknownOp},
			{expr: `foo!!"bar"`, err: ErrUnknownOp},
			{expr: `foo=="bar"`, err: ErrUnknownOp},
		}

		for _, tc := range testCases {
			tm, err := ParseMatcher(tc.expr)
			if tc.err != nil {
				Expect(errors.Is(err, tc.err)).To(BeTrue())
			} else {
				Expect(err).To(BeNil())
			}
			if tm != nil {
				tm.R = nil
			}
			Expect(tm).To(Equal(tc.m))
		}
	})

	It("matchers are properly sorted by operator priority", func() {
		m, err := ParseMatchers(`3="x",4=~"x",1!="x",2!~"x"`)
		Expect(err).ToNot(HaveOccurred())
		x := regexp.MustCompile("x")
		Expect(m).To(Equal([]*TagMatcher{
			{"1", "x", NEQ, nil},
			{"2", "x", NEQ_REGEX, x},
			{"3", "x", EQL, nil},
			{"4", "x", EQL_REGEX, x},
		}))
	})
})

var _ = Describe("ParseMatcherRegex", func() {
	It("parses tag matchers with regex", func() {
		m, err := ParseMatcher(`foo=~".*_suffix"`)
		Expect(err).To(BeNil())
		Expect(m).ToNot(BeNil())

		m, err = ParseMatcher(`foo=~"["`)
		Expect(m).To(BeNil())
		Expect(err).ToNot(BeNil())

		var e1 *syntax.Error
		Expect(errors.As(err, &e1)).To(BeTrue())

		var e2 *Error
		Expect(errors.As(err, &e2)).To(BeTrue())
	})
})
