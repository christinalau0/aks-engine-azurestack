// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package i18n

import (
	"os"
	"strings"
	"testing"

	"github.com/onsi/gomega"
)

func TestLoadTranslations(t *testing.T) {
	gomega.RegisterTestingT(t)

	origLang := os.Getenv("LANG")
	if origLang != "" && !strings.HasPrefix(origLang, "en_US") {
		// The unit test has only en_US translation files
		return
	}

	_, err := LoadTranslations()
	gomega.Expect(err).Should(gomega.BeNil())
}

func TestTranslationLanguage(t *testing.T) {
	gomega.RegisterTestingT(t)

	origLang := os.Getenv("LANG")
	os.Setenv("LANG", "en_US.UTF-8")
	_, err := LoadTranslations()
	gomega.Expect(err).Should(gomega.BeNil())

	lang := GetLanguage()
	gomega.Expect(lang).Should(gomega.Equal("en_US"))

	os.Setenv("LANG", origLang)
}

func TestTranslationLanguageDefault(t *testing.T) {
	gomega.RegisterTestingT(t)

	origLang := os.Getenv("LANG")
	os.Setenv("LANG", "ll_CC.UTF-8")
	_, err := LoadTranslations()
	gomega.Expect(err).Should(gomega.BeNil())

	lang := GetLanguage()
	gomega.Expect(lang).Should(gomega.Equal(defaultLanguage))

	os.Setenv("LANG", origLang)
}

func TestTranslations(t *testing.T) {
	gomega.RegisterTestingT(t)

	origLang := os.Getenv("LANG")
	if origLang != "" && !strings.HasPrefix(origLang, "en_US") {
		// The unit test has only en_US translation files
		return
	}

	l, err := LoadTranslations()
	gomega.Expect(err).Should(gomega.BeNil())

	translator := &Translator{
		Locale: l,
	}

	msg := translator.T("Aloha")
	gomega.Expect(msg).Should(gomega.Equal("Aloha"))

	msg = translator.T("Hello %s", "World")
	gomega.Expect(msg).Should(gomega.Equal("Hello World"))
}

func TestTranslationsPlural(t *testing.T) {
	gomega.RegisterTestingT(t)

	origLang := os.Getenv("LANG")
	if origLang != "" && !strings.HasPrefix(origLang, "en_US") {
		// The unit test has only en_US translation files
		return
	}

	l, err := LoadTranslations()
	gomega.Expect(err).Should(gomega.BeNil())

	translator := &Translator{
		Locale: l,
	}

	msg := translator.NT("There is %d parameter in resource %s", "There are %d parameters in resource %s", 1, 1, "Foo")
	gomega.Expect(msg).Should(gomega.Equal("There is 1 parameter in resource Foo"))

	msg = translator.NT("There is %d parameter in resource %s", "There are %d parameters in resource %s", 9, 9, "Foo")
	gomega.Expect(msg).Should(gomega.Equal("There are 9 parameters in resource Foo"))
}

func TestTranslationsError(t *testing.T) {
	gomega.RegisterTestingT(t)

	origLang := os.Getenv("LANG")
	if origLang != "" && !strings.HasPrefix(origLang, "en_US") {
		// The unit test has only en_US translation files
		return
	}

	l, err := LoadTranslations()
	gomega.Expect(err).Should(gomega.BeNil())

	translator := &Translator{
		Locale: l,
	}

	e := translator.Errorf("File not exists")
	gomega.Expect(e.Error()).Should(gomega.Equal("File not exists"))

	e = translator.NErrorf("There is %d error in the api model", "There are %d errors in the api model", 3, 3)
	gomega.Expect(e.Error()).Should(gomega.Equal("There are 3 errors in the api model"))
}
